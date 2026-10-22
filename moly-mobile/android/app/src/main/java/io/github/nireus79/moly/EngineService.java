package io.github.nireus79.moly;

import android.app.Notification;
import android.app.NotificationChannel;
import android.app.NotificationManager;
import android.app.Service;
import android.content.Intent;
import android.content.pm.ServiceInfo;
import android.os.Build;
import android.os.IBinder;
import android.util.Log;

import java.io.BufferedReader;
import java.io.File;
import java.io.FileWriter;
import java.io.InputStreamReader;
import java.net.HttpURLConnection;
import java.net.URL;
import java.util.Map;

// Owns the engine process. A foreground service keeps the app (and so the engine, its child) at foreground priority
// while a reply is being written, even with the screen off or the app swiped away.
public class EngineService extends Service {
    private static final String TAG = "Moly";
    private static Process engine;
    private static Thread heartbeat;
    private static int enginePort; // the port the running engine was started with
    private static volatile String slowResult = "not started";
    // The engine stops itself (and the "Moly is running" notice goes) when the app has been off the screen this long and the
    // engine has served nothing for as long. The engine's own work after a reply ends within seconds of its request.
    private static final long IDLE_STOP_MS = 3 * 60 * 1000;
    private static volatile boolean appVisible = false; // MainActivity.onStart sets it
    private static volatile int lastStartId;
    private static volatile long lastVisible = System.currentTimeMillis();

    // Called by MainActivity when the app comes onto, or leaves, the screen.
    static void appVisible(boolean visible) {
        appVisible = visible;
        lastVisible = System.currentTimeMillis();
    }

    static boolean engineAlive() {
        return engine != null && engine.isAlive();
    }

    @Override
    public IBinder onBind(Intent i) {
        return null;
    }

    @Override
    public int onStartCommand(Intent intent, int flags, int startId) {
        log(intent == null ? "service started again by Android (it had been stopped)" : "service started");
        lastVisible = System.currentTimeMillis(); // the idle wait counts from now
        lastStartId = startId;
        Notification.Builder nb = new Notification.Builder(this);
        if (Build.VERSION.SDK_INT >= 26) {
            NotificationManager nm = getSystemService(NotificationManager.class);
            nm.createNotificationChannel(new NotificationChannel("engine", "Moly in the background", NotificationManager.IMPORTANCE_LOW));
            nb = new Notification.Builder(this, "engine");
        }
        Notification n = nb.setContentTitle("Moly is running")
                .setContentText("Lets replies finish while you are away; stops by itself when idle")
                .setSmallIcon(android.R.drawable.ic_popup_sync)
                .setOngoing(true)
                .build();
        if (Build.VERSION.SDK_INT >= 29) {
            startForeground(1, n, ServiceInfo.FOREGROUND_SERVICE_TYPE_DATA_SYNC);
        } else {
            startForeground(1, n);
        }
        try {
            startEngine();
        } catch (Throwable t) {
            Log.e(TAG, "engine start failed", t);
        }
        if (intent != null && intent.getStringExtra("slowUrl") != null) {
            final String url = intent.getStringExtra("slowUrl");
            new Thread(() -> slowRequest(url)).start();
        }
        return START_STICKY;
    }

    // Debug builds only: lines KEY=VALUE from files/test-env.txt (for example MOLY_ANTHROPIC_BASE_URL, to point the engine at a
    // stand-in model server in tests). A release build never reads it.
    private void addTestEnvironment(Map<String, String> env) {
        if ((getApplicationInfo().flags & android.content.pm.ApplicationInfo.FLAG_DEBUGGABLE) == 0) return;
        File f = new File(getFilesDir(), "test-env.txt");
        if (!f.exists()) return;
        try (java.io.BufferedReader r = new java.io.BufferedReader(new java.io.FileReader(f))) {
            String line;
            while ((line = r.readLine()) != null) {
                int i = line.indexOf('=');
                if (i > 0) env.put(line.substring(0, i).trim(), line.substring(i + 1).trim());
            }
        } catch (Exception e) {
            Log.e(TAG, "test-env.txt not read", e);
        }
    }

    private synchronized void startEngine() throws Exception {
        int port = EnginePort.get(this);
        if (engine != null && engine.isAlive()) {
            if (port == enginePort) return;
            log("engine port changed to " + port + ": restarting the engine");
            engine.destroy(); // a clean stop (SIGTERM); the engine closes its database
            if (!engine.waitFor(8, java.util.concurrent.TimeUnit.SECONDS)) engine.destroyForcibly();
        }
        File exe = new File(getApplicationInfo().nativeLibraryDir, "libmolyengine.so");
        ProcessBuilder pb = new ProcessBuilder(exe.getAbsolutePath());
        Map<String, String> env = pb.environment();
        env.put("HOME", getFilesDir().getAbsolutePath());
        env.put("MOLY_PRODUCTION", "1");
        env.put("MOLY_PORT", ":" + port);
        env.put("MOLY_DB_KEY", KeyHelper.loadOrCreateDbKey(this));
        addTestEnvironment(env);
        pb.redirectErrorStream(true);
        File engineLog = new File(getFilesDir(), "engine.log");
        trim(engineLog, 1024 * 1024, 256 * 1024);
        pb.redirectOutput(ProcessBuilder.Redirect.appendTo(engineLog));
        final Process started = pb.start();
        engine = started;
        enginePort = port;
        log("engine started on port " + port);
        new Thread(() -> {
            try {
                int code = started.waitFor();
                log("engine exited with code " + code);
            } catch (InterruptedException ignored) {
            }
        }).start();
        if (heartbeat == null) {
            heartbeat = new Thread(this::beat);
            heartbeat.start();
        }
    }

    // A request that stays open for many minutes, like a slow model reply, to see whether Android lets it finish.
    private void slowRequest(String url) {
        long t0 = System.currentTimeMillis();
        slowResult = "running";
        log("slow request started " + url);
        try {
            HttpURLConnection c = (HttpURLConnection) new URL(url).openConnection();
            c.setConnectTimeout(10000);
            c.setReadTimeout(20 * 60 * 1000);
            String line = new BufferedReader(new InputStreamReader(c.getInputStream())).readLine();
            slowResult = "done after " + (System.currentTimeMillis() - t0) / 1000 + " s: " + line;
        } catch (Throwable t) {
            slowResult = "FAILED after " + (System.currentTimeMillis() - t0) / 1000 + " s: " + t;
        }
        log("slow request " + slowResult);
    }

    private void beat() {
        while (true) {
            String status = get(EnginePort.url(this) + "/api/status");
            log("beat engineAlive=" + engineAlive() + " status=" + status + (slowResult.equals("not started") ? "" : " slow=" + slowResult));
            if (stopWhenIdle()) return;
            try {
                Thread.sleep(60000);
            } catch (InterruptedException e) {
                return;
            }
        }
    }

    // Stops the engine and this service when the app has been off the screen for IDLE_STOP_MS and the engine reports that
    // long without a request. Any doubt (no answer, no token, an unreadable answer) keeps it running. Opening the app again
    // starts the engine anew (MolyEnginePlugin.start). Returns true when it stopped.
    private boolean stopWhenIdle() {
        if (appVisible || System.currentTimeMillis() - lastVisible < IDLE_STOP_MS || !engineAlive()) return false;
        long idle = engineIdleSeconds();
        if (idle < IDLE_STOP_MS / 1000) return false;
        log("idle for " + idle + " s with the app off the screen: stopping the engine");
        synchronized (EngineService.class) {
            if (appVisible) return false; // the person came back while we were deciding
            Process p = engine;
            if (p != null) {
                p.destroy(); // a clean stop (SIGTERM); the engine closes its database
                try {
                    if (!p.waitFor(16, java.util.concurrent.TimeUnit.SECONDS)) p.destroyForcibly();
                } catch (InterruptedException ignored) {
                    p.destroyForcibly();
                }
            }
            heartbeat = null;
        }
        stopForeground(true);
        stopSelf(lastStartId); // only if nobody has asked the service to start again meanwhile
        return true;
    }

    // Seconds since the engine last served a request (0 while one is running); -1 when that cannot be read.
    private long engineIdleSeconds() {
        try {
            File tokenFile = new File(getFilesDir(), ".moly/session.token");
            String token = new BufferedReader(new java.io.FileReader(tokenFile)).readLine();
            HttpURLConnection c = (HttpURLConnection) new URL(EnginePort.url(this) + "/internal/idle").openConnection();
            c.setConnectTimeout(3000);
            c.setReadTimeout(3000);
            c.setRequestProperty("X-Moly-Token", token == null ? "" : token.trim());
            String body = new BufferedReader(new InputStreamReader(c.getInputStream())).readLine();
            java.util.regex.Matcher m = java.util.regex.Pattern.compile("\"idleSeconds\":(\\d+)").matcher(body == null ? "" : body);
            return m.find() ? Long.parseLong(m.group(1)) : -1;
        } catch (Throwable t) {
            return -1;
        }
    }

    private String get(String u) {
        try {
            HttpURLConnection c = (HttpURLConnection) new URL(u).openConnection();
            c.setConnectTimeout(5000);
            c.setReadTimeout(5000);
            int code = c.getResponseCode();
            return String.valueOf(code);
        } catch (Throwable t) {
            return "ERR " + t.getClass().getSimpleName();
        }
    }

    private void log(String s) {
        String line = new java.text.SimpleDateFormat("MM-dd HH:mm:ss").format(new java.util.Date()) + " " + s;
        Log.i(TAG, line);
        File f = new File(getFilesDir(), "heartbeat.log");
        trim(f, 200 * 1024, 100 * 1024);
        try (FileWriter w = new FileWriter(f, true)) {
            w.write(line + "\n");
        } catch (Exception ignored) {
        }
    }

    // A log that grows past `max` bytes keeps only its last `keep` bytes (from the next whole line).
    private static synchronized void trim(File f, long max, long keep) {
        if (!f.exists() || f.length() <= max) return;
        try (java.io.RandomAccessFile r = new java.io.RandomAccessFile(f, "r")) {
            r.seek(f.length() - keep);
            r.readLine();
            byte[] rest = new byte[(int) (r.length() - r.getFilePointer())];
            r.readFully(rest);
            try (java.io.FileOutputStream o = new java.io.FileOutputStream(f)) {
                o.write(rest);
            }
        } catch (Exception ignored) {
        }
    }

    // Android 15 and later limit a data-sync foreground service to about six hours a day and then call this; the service must stop
    // itself at once (the engine closes its database on a clean stop). Opening Moly again starts it anew.
    @Override
    public void onTimeout(int startId, int fgsType) {
        log("Android's time limit for a background service was reached: stopping the engine");
        synchronized (EngineService.class) {
            Process p = engine;
            if (p != null) {
                p.destroy();
                try {
                    if (!p.waitFor(8, java.util.concurrent.TimeUnit.SECONDS)) p.destroyForcibly();
                } catch (InterruptedException ignored) {
                    p.destroyForcibly();
                }
            }
            heartbeat = null;
        }
        stopForeground(true);
        stopSelf(startId);
    }

    @Override
    public void onTaskRemoved(Intent rootIntent) {
        log("app swiped away from the recent apps");
        super.onTaskRemoved(rootIntent);
    }

    @Override
    public void onDestroy() {
        log("service destroyed");
        super.onDestroy();
    }
}

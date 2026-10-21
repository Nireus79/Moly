package io.github.nireus79.moly;

import android.app.Activity;
import android.content.Context;
import android.content.Intent;
import android.net.Uri;
import android.os.Build;

import com.getcapacitor.JSObject;
import com.getcapacitor.Plugin;
import com.getcapacitor.PluginCall;
import com.getcapacitor.PluginMethod;
import com.getcapacitor.annotation.ActivityCallback;
import com.getcapacitor.annotation.CapacitorPlugin;
import androidx.activity.result.ActivityResult;

import java.io.BufferedReader;
import java.io.File;
import java.io.FileInputStream;
import java.io.ByteArrayOutputStream;
import java.io.InputStream;
import java.io.InputStreamReader;
import java.io.OutputStream;
import java.nio.charset.StandardCharsets;
import java.net.HttpURLConnection;
import java.net.URL;

// The phone's replacement for the native messaging host: starts the engine (in the foreground service) and hands the
// screens its address and session token.
@CapacitorPlugin(name = "MolyEngine")
public class MolyEnginePlugin extends Plugin {
    private static final int MAX_FILE_BYTES = 64 * 1024 * 1024;
    private String pendingSave; // the text to write once the person has chosen where
    private static String sharedText; // text another app shared to Moly; held in memory only, until the screen takes it

    static void setShared(String text, com.getcapacitor.Bridge bridge) {
        sharedText = text;
        if (bridge != null) bridge.triggerWindowJSEvent("molyShared", "{}");
    }

    // Which store build this is and what its ad slot is (a stand-in for the ad and billing code of the Play build).
    @PluginMethod
    public void flavorInfo(PluginCall call) {
        JSObject r = new JSObject();
        r.put("slot", AdSlot.describe());
        r.put("appId", getContext().getPackageName());
        call.resolve(r);
    }

    @PluginMethod
    public void getSharedText(PluginCall call) {
        JSObject r = new JSObject();
        r.put("text", sharedText);
        sharedText = null;
        call.resolve(r);
    }

    // "Save a copy" of the person's data: Android's "save as" picker, so they choose where the file goes.
    @PluginMethod
    public void saveFile(PluginCall call) {
        pendingSave = call.getString("text", "");
        Intent i = new Intent(Intent.ACTION_CREATE_DOCUMENT);
        i.addCategory(Intent.CATEGORY_OPENABLE);
        i.setType("application/json");
        i.putExtra(Intent.EXTRA_TITLE, call.getString("name", "moly-copy.json"));
        startActivityForResult(call, i, "saveFileResult");
    }

    @ActivityCallback
    private void saveFileResult(PluginCall call, ActivityResult result) {
        if (call == null) return;
        String text = pendingSave;
        pendingSave = null;
        Uri uri = result.getResultCode() == Activity.RESULT_OK && result.getData() != null ? result.getData().getData() : null;
        JSObject r = new JSObject();
        if (uri == null || text == null) {
            r.put("saved", false); // the person cancelled
            call.resolve(r);
            return;
        }
        try (OutputStream o = getContext().getContentResolver().openOutputStream(uri, "wt")) {
            if (o == null) throw new java.io.IOException("no output stream");
            o.write(text.getBytes(StandardCharsets.UTF_8));
            r.put("saved", true);
            call.resolve(r);
        } catch (Exception e) {
            call.reject("Could not write the file: " + e.getMessage());
        }
    }

    // "Restore from a copy": Android's file picker; the text of the chosen file goes to the screens.
    @PluginMethod
    public void pickFile(PluginCall call) {
        Intent i = new Intent(Intent.ACTION_OPEN_DOCUMENT);
        i.addCategory(Intent.CATEGORY_OPENABLE);
        i.setType("*/*");
        i.putExtra(Intent.EXTRA_MIME_TYPES, new String[]{"application/json", "text/plain", "application/octet-stream"});
        startActivityForResult(call, i, "pickFileResult");
    }

    @ActivityCallback
    private void pickFileResult(PluginCall call, ActivityResult result) {
        if (call == null) return;
        Uri uri = result.getResultCode() == Activity.RESULT_OK && result.getData() != null ? result.getData().getData() : null;
        JSObject r = new JSObject();
        if (uri == null) {
            r.put("text", JSObject.NULL); // the person cancelled
            call.resolve(r);
            return;
        }
        try (InputStream in = getContext().getContentResolver().openInputStream(uri)) {
            if (in == null) throw new java.io.IOException("no input stream");
            ByteArrayOutputStream buf = new ByteArrayOutputStream();
            byte[] chunk = new byte[64 * 1024];
            int n;
            while ((n = in.read(chunk)) > 0) {
                buf.write(chunk, 0, n);
                if (buf.size() > MAX_FILE_BYTES) {
                    call.reject("The file is too large to be a Moly copy.");
                    return;
                }
            }
            r.put("text", new String(buf.toByteArray(), StandardCharsets.UTF_8));
            call.resolve(r);
        } catch (Exception e) {
            call.reject("Could not read the file: " + e.getMessage());
        }
    }

    // The report text, built on the phone; nothing leaves it until shareReport.
    @PluginMethod
    public void buildReport(PluginCall call) {
        new Thread(() -> {
            String status = get("/api/status");
            String version = status == null ? "" : status.replaceAll(".*\"version\":\"([^\"]*)\".*", "$1");
            JSObject r = new JSObject();
            r.put("text", Report.build(getContext(), version));
            call.resolve(r);
        }).start();
    }

    // The phone's share sheet with the text the person has just read; they choose where it goes.
    @PluginMethod
    public void shareReport(PluginCall call) {
        String text = call.getString("text", "");
        Intent send = new Intent(Intent.ACTION_SEND);
        send.setType("text/plain");
        send.putExtra(Intent.EXTRA_SUBJECT, "Moly report");
        send.putExtra(Intent.EXTRA_TEXT, text);
        Intent chooser = Intent.createChooser(send, "Send the Moly report");
        chooser.addFlags(Intent.FLAG_ACTIVITY_NEW_TASK);
        getContext().startActivity(chooser);
        call.resolve();
    }

    @PluginMethod
    public void start(PluginCall call) {
        Context ctx = getContext();
        int port = call.getInt("port", 0);
        if (port != 0) {
            if (port < 1024 || port > 65535) {
                call.reject("Not a usable port: " + port);
                return;
            }
            EnginePort.set(ctx, port);
        }
        Intent svc = new Intent(ctx, EngineService.class);
        if (Build.VERSION.SDK_INT >= 26) ctx.startForegroundService(svc); else ctx.startService(svc);
        try {
            String status = null;
            for (int i = 0; i < 120 && status == null; i++) {
                status = get("/api/status");
                // An engine that is not running after the first seconds could not take its port (another program has it).
                if (status == null && i >= 8 && !EngineService.engineAlive()) {
                    call.reject("ENGINE_STOPPED: the engine could not start on port " + EnginePort.get(ctx));
                    return;
                }
                if (status == null) Thread.sleep(500);
            }
            if (status == null) {
                call.reject("The engine did not answer in 60 seconds.");
                return;
            }
            File tokenFile = new File(ctx.getFilesDir(), ".moly/session.token");
            String token = new BufferedReader(new InputStreamReader(new FileInputStream(tokenFile))).readLine();
            JSObject r = new JSObject();
            r.put("url", EnginePort.url(ctx));
            r.put("token", token);
            r.put("version", status.replaceAll(".*\"version\":\"([^\"]*)\".*", "$1"));
            call.resolve(r);
        } catch (Throwable t) {
            call.reject("Could not start the engine: " + t);
        }
    }

    private String get(String path) {
        try {
            HttpURLConnection c = (HttpURLConnection) new URL(EnginePort.url(getContext()) + path).openConnection();
            c.setConnectTimeout(1000);
            c.setReadTimeout(5000);
            return new BufferedReader(new InputStreamReader(c.getInputStream())).readLine();
        } catch (Exception e) {
            return null;
        }
    }
}

package io.github.nireus79.moly;

import android.content.Context;
import android.content.Intent;
import android.os.Build;

import com.getcapacitor.JSObject;
import com.getcapacitor.Plugin;
import com.getcapacitor.PluginCall;
import com.getcapacitor.PluginMethod;
import com.getcapacitor.annotation.CapacitorPlugin;

import java.io.BufferedReader;
import java.io.File;
import java.io.FileInputStream;
import java.io.InputStreamReader;
import java.net.HttpURLConnection;
import java.net.URL;

// The phone's replacement for the native messaging host: starts the engine (in the foreground service) and hands the
// screens its address and session token.
@CapacitorPlugin(name = "MolyEngine")
public class MolyEnginePlugin extends Plugin {
    private static final String URL = "http://127.0.0.1:11436";
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
        Intent svc = new Intent(ctx, EngineService.class);
        if (Build.VERSION.SDK_INT >= 26) ctx.startForegroundService(svc); else ctx.startService(svc);
        try {
            String status = null;
            for (int i = 0; i < 120 && status == null; i++) {
                status = get("/api/status");
                if (status == null) Thread.sleep(500);
            }
            if (status == null) {
                call.reject("The engine did not answer in 60 seconds.");
                return;
            }
            File tokenFile = new File(ctx.getFilesDir(), ".moly/session.token");
            String token = new BufferedReader(new InputStreamReader(new FileInputStream(tokenFile))).readLine();
            JSObject r = new JSObject();
            r.put("url", URL);
            r.put("token", token);
            r.put("version", status.replaceAll(".*\"version\":\"([^\"]*)\".*", "$1"));
            call.resolve(r);
        } catch (Throwable t) {
            call.reject("Could not start the engine: " + t);
        }
    }

    private String get(String path) {
        try {
            HttpURLConnection c = (HttpURLConnection) new URL(URL + path).openConnection();
            c.setConnectTimeout(1000);
            c.setReadTimeout(5000);
            return new BufferedReader(new InputStreamReader(c.getInputStream())).readLine();
        } catch (Exception e) {
            return null;
        }
    }
}

package io.github.nireus79.moly;

import android.content.Context;

import java.io.File;
import java.io.FileWriter;

// Writes one line to the event log when the app itself crashes (the class of the error and where, never its message), then
// lets Android handle the crash as before, so Google Play still receives it. The line shows in "Report a problem".
final class CrashNote {
    private static boolean installed = false;

    private CrashNote() {}

    static synchronized void install(Context ctx) {
        if (installed) return;
        installed = true;
        final File log = new File(ctx.getApplicationContext().getFilesDir(), "heartbeat.log");
        final Thread.UncaughtExceptionHandler previous = Thread.getDefaultUncaughtExceptionHandler();
        Thread.setDefaultUncaughtExceptionHandler((thread, t) -> {
            try (FileWriter w = new FileWriter(log, true)) {
                w.write(new java.text.SimpleDateFormat("MM-dd HH:mm:ss").format(new java.util.Date()) + " " + CrashText.describeCrash(t, 6) + "\n");
            } catch (Throwable ignored) {
            }
            if (previous != null) previous.uncaughtException(thread, t);
        });
    }
}

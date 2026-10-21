package io.github.nireus79.moly;

import android.content.Context;
import android.content.SharedPreferences;

// The loopback port of this install's engine. The screens pick a random port once and keep it (so another app, or another
// copy of Moly, cannot be sitting on it); the plugin saves what the screens ask for, and the service uses it, also when
// Android restarts the service without them.
final class EnginePort {
    static final int FALLBACK = 11436;

    private EnginePort() {}

    static int get(Context ctx) {
        return prefs(ctx).getInt("enginePort", FALLBACK);
    }

    static void set(Context ctx, int port) {
        prefs(ctx).edit().putInt("enginePort", port).apply();
    }

    static String url(Context ctx) {
        return "http://127.0.0.1:" + get(ctx);
    }

    private static SharedPreferences prefs(Context ctx) {
        return ctx.getSharedPreferences("moly", Context.MODE_PRIVATE);
    }
}

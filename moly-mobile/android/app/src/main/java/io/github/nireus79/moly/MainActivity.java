package io.github.nireus79.moly;

import android.Manifest;
import android.content.Intent;
import android.content.SharedPreferences;
import android.os.Build;
import android.os.Bundle;
import android.util.Log;

import com.getcapacitor.BridgeActivity;

public class MainActivity extends BridgeActivity {
    // How long the app was off the screen before it last came back (0 when not known). Kept on the phone by onStop, so it
    // survives the app being killed or force-stopped while away. The screens ask for it (MolyEnginePlugin.away) and are told
    // when the app returns (the window event molyReturned), because the web view's own visibility events are not reliable.
    static volatile long lastAway = 0;

    private SharedPreferences awayStore() {
        return getSharedPreferences("moly_away", MODE_PRIVATE);
    }

    @Override
    public void onCreate(Bundle savedInstanceState) {
        registerPlugin(MolyEnginePlugin.class);
        super.onCreate(savedInstanceState);
        handleShare(getIntent());
        if (Build.VERSION.SDK_INT >= 33) requestPermissions(new String[]{Manifest.permission.POST_NOTIFICATIONS}, 1);
    }

    @Override
    public void onStart() {
        super.onStart();
        EngineService.appVisible(true);
        long left = awayStore().getLong("leftAt", 0);
        lastAway = left > 0 ? Math.max(0, System.currentTimeMillis() - left) : 0;
        awayStore().edit().remove("leftAt").commit();
        Log.i("Moly", "app on screen again after " + lastAway / 1000 + " s away");
        if (getBridge() != null) getBridge().triggerWindowJSEvent("molyReturned"); // lost, harmlessly, while the screens are still loading
    }

    @Override
    public void onStop() {
        EngineService.appVisible(false);
        awayStore().edit().putLong("leftAt", System.currentTimeMillis()).commit();
        super.onStop();
    }

    @Override
    protected void onNewIntent(Intent intent) {
        super.onNewIntent(intent);
        handleShare(intent);
    }

    // "Share to Moly": text only, handed to the screen when the person shares it. Nothing is read otherwise.
    private void handleShare(Intent intent) {
        if (intent != null && Intent.ACTION_SEND.equals(intent.getAction()) && "text/plain".equals(intent.getType())) {
            String text = intent.getStringExtra(Intent.EXTRA_TEXT);
            if (text != null && !text.isEmpty()) MolyEnginePlugin.setShared(text, getBridge());
        }
    }
}

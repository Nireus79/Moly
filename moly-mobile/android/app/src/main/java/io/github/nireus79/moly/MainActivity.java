package io.github.nireus79.moly;

import android.Manifest;
import android.content.Intent;
import android.os.Build;
import android.os.Bundle;

import com.getcapacitor.BridgeActivity;

public class MainActivity extends BridgeActivity {
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
    }

    @Override
    public void onStop() {
        EngineService.appVisible(false);
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

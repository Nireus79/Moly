package io.github.nireus79.moly;

import android.content.Context;
import android.content.SharedPreferences;
import android.security.keystore.KeyGenParameterSpec;
import android.security.keystore.KeyProperties;
import android.util.Base64;

import java.security.KeyStore;
import java.security.SecureRandom;

import javax.crypto.Cipher;
import javax.crypto.KeyGenerator;
import javax.crypto.SecretKey;
import javax.crypto.spec.GCMParameterSpec;

final class KeyHelper {
    private KeyHelper() {}

    // A random 32-byte key for the database, stored encrypted with a key that never leaves the Android Keystore.
    static String loadOrCreateDbKey(Context ctx) throws Exception {
        KeyStore ks = KeyStore.getInstance("AndroidKeyStore");
        ks.load(null);
        if (!ks.containsAlias("moly-db-wrap")) {
            KeyGenerator kg = KeyGenerator.getInstance(KeyProperties.KEY_ALGORITHM_AES, "AndroidKeyStore");
            kg.init(new KeyGenParameterSpec.Builder("moly-db-wrap", KeyProperties.PURPOSE_ENCRYPT | KeyProperties.PURPOSE_DECRYPT)
                    .setBlockModes(KeyProperties.BLOCK_MODE_GCM)
                    .setEncryptionPaddings(KeyProperties.ENCRYPTION_PADDING_NONE)
                    .setKeySize(256).build());
            kg.generateKey();
        }
        SecretKey wrap = (SecretKey) ks.getKey("moly-db-wrap", null);
        SharedPreferences sp = ctx.getSharedPreferences("moly", Context.MODE_PRIVATE);
        String stored = sp.getString("dbkey", null);
        byte[] key;
        if (stored == null) {
            key = new byte[32];
            new SecureRandom().nextBytes(key);
            Cipher c = Cipher.getInstance("AES/GCM/NoPadding");
            c.init(Cipher.ENCRYPT_MODE, wrap);
            byte[] ct = c.doFinal(key);
            sp.edit().putString("dbkey", Base64.encodeToString(c.getIV(), Base64.NO_WRAP) + ":" + Base64.encodeToString(ct, Base64.NO_WRAP)).apply();
        } else {
            String[] p = stored.split(":");
            Cipher c = Cipher.getInstance("AES/GCM/NoPadding");
            c.init(Cipher.DECRYPT_MODE, wrap, new GCMParameterSpec(128, Base64.decode(p[0], Base64.NO_WRAP)));
            key = c.doFinal(Base64.decode(p[1], Base64.NO_WRAP));
        }
        StringBuilder sb = new StringBuilder();
        for (byte x : key) sb.append(String.format("%02x", x));
        return sb.toString();
    }
}

package io.github.nireus79.moly;

final class AdSlot {
    private AdSlot() {}

    static String describe() {
        // Stand-in for the ad network and Play Billing: a Google Play services class that only this build carries.
        return "play: " + com.google.android.gms.ads.identifier.AdvertisingIdClient.class.getSimpleName() + " present";
    }
}

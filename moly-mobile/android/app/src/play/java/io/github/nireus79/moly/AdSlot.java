package io.github.nireus79.moly;

final class AdSlot {
    private AdSlot() {}

    static String describe() {
        // The first Play release carries no ad network and no billing library (the free app with its sponsor card only).
        return "play: no ad or billing library";
    }
}

package io.github.nireus79.moly;

import java.util.ArrayList;
import java.util.List;
import java.util.regex.Pattern;

// Pure text helpers for crash visibility (no Android classes, so they are tested on the computer).
final class CrashText {
    private CrashText() {}

    private static final Pattern START = Pattern.compile("^(panic:|fatal error:|SIGSEGV|unexpected fault address)");
    private static final Pattern FRAME = Pattern.compile("^(goroutine \\d+|[A-Za-z0-9_./()*-]+\\.[A-Za-z0-9_()*.-]+\\(.*\\)$|\\s+\\S+\\.go:\\d+)");

    // The last Go crash in an engine log: its kind (the first words of the first line, cut at 60 characters, because a panic value can
    // be any text) and then only the lines that name a function or a source file and line, at most `maxFrames`. Returns an empty list
    // when the log holds no crash. The panic's own message text is not kept beyond the kind.
    static List<String> lastGoCrash(List<String> lines, int maxFrames) {
        int at = -1;
        for (int i = 0; i < lines.size(); i++) if (START.matcher(lines.get(i)).find()) at = i;
        List<String> out = new ArrayList<>();
        if (at < 0) return out;
        String first = lines.get(at);
        int colon = first.indexOf(':');
        String kind = colon > 0 && colon < 12 ? first.substring(0, colon + 1) : first;
        String rest = colon > 0 && colon < 12 ? first.substring(colon + 1).trim() : "";
        out.add(kind + (rest.isEmpty() ? "" : " " + (rest.length() > 60 ? rest.substring(0, 60) + "..." : rest)));
        int frames = 0;
        for (int i = at + 1; i < lines.size() && frames < maxFrames; i++) {
            String l = lines.get(i);
            if (FRAME.matcher(l).find()) {
                out.add(l.trim());
                frames++;
            }
        }
        return out;
    }

    // A crash of the app itself as one line for the event log: the exception class and the first frames of this app, never the
    // message (it can hold anything the person typed).
    static String describeCrash(Throwable t, int maxFrames) {
        StringBuilder b = new StringBuilder("app crashed: ").append(t.getClass().getName());
        int n = 0;
        for (StackTraceElement e : t.getStackTrace()) {
            if (n >= maxFrames) break;
            b.append(n == 0 ? " at " : " < ").append(e.getClassName()).append('.').append(e.getMethodName()).append(':').append(e.getLineNumber());
            n++;
        }
        return b.toString();
    }
}

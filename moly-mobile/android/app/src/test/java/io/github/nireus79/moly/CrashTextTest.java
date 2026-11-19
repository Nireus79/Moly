package io.github.nireus79.moly;

import static org.junit.Assert.*;

import java.util.Arrays;
import java.util.List;
import org.junit.Test;

public class CrashTextTest {
    @Test
    public void noCrashGivesNothing() {
        assertTrue(CrashText.lastGoCrash(Arrays.asList("2026/11/19 [Moly] ready", "[Moly] stop requested"), 10).isEmpty());
    }

    @Test
    public void aPanicGivesItsKindAndOnlyFrames() {
        List<String> log = Arrays.asList(
                "2026/11/19 [Moly] ready",
                "panic: runtime error: index out of range [3] with length 2",
                "",
                "goroutine 42 [running]:",
                "moly/agents.(*ConversationAgent).reply(0xc0001, {0x1})",
                "\t/src/moly-go/agents/conversation_agent.go:1136 +0x2a",
                "some other chatter from the person's message: hello there",
                "main.main()",
                "\t/src/moly-go/main.go:260 +0x10");
        List<String> got = CrashText.lastGoCrash(log, 10);
        assertEquals("panic: runtime error: index out of range [3] with length 2", got.get(0));
        assertTrue(got.contains("goroutine 42 [running]:"));
        assertTrue(got.contains("/src/moly-go/agents/conversation_agent.go:1136 +0x2a"));
        for (String l : got) assertFalse("a line of other text leaked: " + l, l.contains("hello there"));
    }

    @Test
    public void aLongPanicValueIsCutAndFramesAreCapped() {
        StringBuilder v = new StringBuilder("panic: ");
        for (int i = 0; i < 50; i++) v.append("secretword ");
        List<String> log = new java.util.ArrayList<>();
        log.add(v.toString());
        for (int i = 0; i < 30; i++) log.add("pkg.f" + i + "()");
        List<String> got = CrashText.lastGoCrash(log, 5);
        assertTrue(got.get(0).length() < 80);
        assertEquals(6, got.size());
    }

    @Test
    public void theLastCrashWins() {
        List<String> got = CrashText.lastGoCrash(Arrays.asList("panic: one", "a.b()", "panic: two", "c.d()"), 10);
        assertEquals("panic: two", got.get(0));
        assertEquals("c.d()", got.get(1));
    }

    @Test
    public void anAppCrashNamesTheClassAndFramesNeverTheMessage() {
        Throwable t = new IllegalStateException("the person typed: my password is hunter2");
        String line = CrashText.describeCrash(t, 3);
        assertTrue(line.startsWith("app crashed: java.lang.IllegalStateException at "));
        assertFalse(line.contains("hunter2"));
        assertFalse(line.contains("person typed"));
    }
}

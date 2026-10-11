# Model portability: does Moly behave the same on every model?

Status: started 2026-10-11. Question from the user: changing the model may change Moly's behaviour; how, and what must the prompting do about it (one prompt set for all models, or one per model)? Quality must not drop.

## 1. What we found so far (evidence from this project)

### 1.1 Two models, two behaviours
| | Mistral 7B (ollama, production target) | Claude Haiku (`claude-cli`, test stand-in) |
|---|---|---|
| Speed | one extraction call 3 min 46 s (long prompt), 2 min 7 s (short prompt); a turn of ~10 calls takes 10-18 min; 100% CPU, 8 cores, 13 GB | ~6 s per call; a turn ~57 s |
| Temperature | set per call (0.1-0.7 depending on the call) | cannot be set through `claude -p`; CLI default, so noisier than production |
| Output habit | plain JSON mostly | wraps JSON in code fences, adds chatter (made the extractor return "nothing" 4 times until `SafeJSONParse`) |
| Long extraction prompt | not measured for quality (too slow) | right 20/30 on goal+person; 0/3 for "I need to write to my manager" (deterministic, so a prompt problem, not noise) |
| Short extraction prompt | not measured for quality | right 27/30 |
| Instruction following | earlier work: a list prompt answered "none" to plain answers in 6 of 12 replays; a keyed-object prompt fixed it (20/20) | followed the "a name states no person" wording literally and returned no person for "Her name is Dana." (the wording was our bug; a model that reads literally exposes it) |
| Safety confirmation | -- | one sampled "no" dropped an explicit self-harm statement (fixed: two "no" needed) |

Reading: the same prompt produced different failures on the two models. Some failures were in our prompt (the wording bug) and only showed on the model that follows instructions literally. Some were format habits (fences). Some were noise from temperature. We have not separated them yet.

### 1.2 What a turn costs (call count)
10-12 sequential model calls per message: meta-instruction check, safety evaluation, extraction, subject attribution (when traits exist), intent, gap generator, clarity analysis, principle reasoner (some turns), reply, insight extraction (feeds the Reflections panel), safety evaluation of the reply, and the background summary. Conditional judges (goal relation, pending answers, person reference, persistence, deepening) run when their case arises. Details and candidates for reduction: `docs/ORCHESTRATOR_DESIGN.md`, "Fixes after the Haiku run", and the call table in this session's report (to be copied here when decided).

### 1.3 Prompt length is a speed and a quality variable
On CPU Mistral, cost follows prompt and answer length: the short extraction prompt (2,702 chars) took 56% of the time of the long one (5,308 chars); its answer was also shorter (767 vs 996 chars). On Haiku the shorter prompt was also more accurate. Whether it is more accurate on Mistral is unknown.

### 1.4 What we cannot yet say
- Whether any prompt is better on Mistral (no quality measurement exists; the one battery run produced no output and was lost).
- How much of Haiku's variance is temperature and how much is prompt length (planned: 10 repeats per case on Haiku; a case that always fails is a prompt problem, one that flips is noise).
- Whether other models (a larger local one, another vendor) would show a third pattern.

### 1.5 Constraints of this machine
Mistral on CPU makes sampling batteries impractical (about 25 hours for 500 calls). Any plan must use few Mistral calls, chosen to be discriminating, and a fast stand-in for the rest.

## 2. Industry practice (research, 2026-10-11)
Sources are mostly practitioner blogs and a few papers; strength is marked. Nothing here was verified by us yet.

1. **Prompts are tuned to a model, not portable text** (practitioner view: tianpan.co "Model Portability Tax", AWS "Model Agility" migration guide, DZone). The recommended architecture keeps the model-specific prompt layer (format, wrappers) apart from business logic, and treats a model change as a migration with an evaluation, not a configuration switch. AWS estimates two days to two weeks per migration.
2. **Format alone can swing a model's results a lot, and models differ in how** (paper: Sclar et al., "Quantifying Language Models' Sensitivity to Spurious Features in Prompt Design", arXiv 2310.11324). Up to 76 accuracy points on LLaMA-2-13B from formatting differences; the sensitivity stayed with bigger models, more examples and instruction tuning; the best format on one model correlates only weakly with the best on another. A Microsoft study found GPT-4-turbo more resilient than earlier models. No study found for sub-7B models specifically. So "the same prompt behaves the same" is not a safe assumption, and small models are the likelier victims.
3. **Fewer, simpler, well-placed instructions for small models** (weak evidence: small-model papers on instruction placement and count, a 7B playbook: "given three requirements the model fulfils one, given five it struggles"). Not settled whether few-shot examples beat rules at 7B.
4. **Structured output is solved at the syntax level, not at the meaning level** (papers: "Constrained Decoding Eliminates Structural Failures in Small LLMs but Reveals a Scale-Dependent Semantic Gap", "The Constraint Tax"). A JSON schema passed to Ollama (`format`) makes output parse; one 7B model went from 0% to 75% valid-and-correct. It does not fix a misunderstanding; a valid object can hold the wrong decision; too strict a schema makes the model invent values, so allow null.
5. **Regression testing on a fixed, versioned set of real cases is the standard gate for a model swap** (practitioner guidance: futureagi, PromptLayer, Tessl). 50-200 cases from production edge cases, scored on several dimensions (accuracy, format, tone, safety), thresholds on the average (not exact match), baseline comparison, re-run on every model or prompt change. Pin the model ID, temperature 0 and a seed for before/after comparison.
6. **Statistics** (guidelines for LLM studies, arXiv 2508.15503; practitioner write-ups): repeat runs and report spread; paired design on the same cases; McNemar's test for pass/fail on the same items, or a bootstrap; 100 cases only detect a 10-12 point difference at 80% power; when run-to-run variance is small, more cases tell more than more repeats; the serving setup (hosting, quantisation) changes results too.
7. **Per-model optimisation is the common workflow for prompt compilers such as DSPy** (secondary sources, no primary documentation read): an optimised prompt is produced for one model and the optimisation is repeated for each model.

### What this means for Moly (reading, not proof)
- Two kinds of model dependence, to be handled differently:
  a. **Judgements** (extraction, intent, safety, goal relation, pending answers, person reference, repeated gap, adjusts-draft, deepening, withdrawal, persistence): small outputs, a right answer exists, so they can be scored objectively against labelled cases. Moly's design already keeps decisions in code (`DecideReply` is a pure function over facts) and unreadable output fails safe, so a weak judgement degrades one fact, not the control flow.
  b. **Free text** (the reply, the summary, drafts): no single right answer. Check properties (one question, goal quoted, no invented facts, no placeholder, in the user's voice, summary keeps the people) and use a judge model or a rubric for the rest.
- The usual answer is neither "one prompt for all" nor "a prompt per model" but a shared prompt core with a thin per-model profile (JSON mode/schema, temperature, length budget, wrapper format) and content differences only where measurement shows a gap beyond noise.

## 3. Do we need to check in practice, and how
Yes: the literature says portability must be measured per task, and our own run already showed model-specific failures (section 1.1). The cost constraint (Mistral about 2-4 min per extraction call on this CPU) means a design, not a brute force.

Proposed design (not started):
1. **A labelled case set per judgement**, taken from what we already have: the dynamic scenario messages (D1-D16), the scripted scenarios (S1-S48) and the failures found. Target about 40-60 cases per judgement that matters most (extraction first, then intent, pending answers, repeated gap, person reference, safety). Each case stores the input (message, exchange, packet) and the expected facts. This is the golden set; it is versioned in the repo.
2. **Scoring by facts, not text**: goal present and quoted, person present with name/label/role, no invented goal or person, valid JSON on the first try (counts format failures separately), safety verdict. A pass rate per judgement and model.
3. **Cheap runs**: temperature 0 and a fixed seed on Mistral (ollama accepts both) so that one run per case is enough and more cases replace repeats; Haiku through the CLI has no temperature control, so it gets 3-5 repeats and is used only to find fragile cases and prompt problems quickly. Small judges have short prompts and should take seconds to tens of seconds on Mistral; that is to be measured (only extraction was timed).
4. **Compare paired**: same cases, model A vs model B, and prompt A vs prompt B; McNemar or bootstrap intervals; report only differences larger than the noise (measured by repeating a small subset).
5. **Decide from the result**: if a prompt is within noise on all models, one prompt; if one model fails where another passes on the same prompt, first try the profile layer (JSON schema, format, length), then a model-specific wording for that judgement only.
6. **Free text checks** use property tests on the reply and summary (some already exist in the scripted harness) plus a small human-read sample from each model, because there is no right answer to score.
7. **Re-run** the set whenever the model, its quantisation, or a prompt changes.

Open questions for the user before building: which models matter besides Mistral (a larger local model, another vendor), and how much overnight machine time is acceptable.

## 4. Decision (2026-10-11): scope is Haiku-level models
The user's scope: Moly targets models around Haiku level (Anthropic, OpenAI, or Ollama with a comparable model); a model as small as Mistral 7B on CPU is too slow for real use (10+ minutes per answer), so per-model quality measurement is deferred. One shared prompt set; a per-model profile only if a real failure shows up on a specific model.

Done:
- The short extraction prompt is the default (`MOLY_EXTRACT_PROMPT=long` selects the old one).
- `LLMRequest.JSONMode`: Ollama `format: "json"`, OpenAI `response_format: json_object`; used by the extractor. Anthropic is not covered (the parser handles fences).
- Live check `TestLiveExtractionInventsNothingFromAMessageWithNoGoal` (off by default, `MOLY_LIVE_LLM=1`): 8 goalless messages x 3 runs on Haiku, 24/24 invented nothing.

Found and fixed: `callOllama` sent `temperature` at the top level of the request; Ollama reads it only from `options`. Checked live on mistral: four calls at temperature 0 gave four different sentences with the top-level field and four identical ones with `options.temperature`. Every Ollama call before this fix ran at Ollama's default temperature. Test: `tools/llm_ollama_test.go`.

# Clawrise OSS Roadmap

## 1. What This Document Is For

This roadmap is written for users, contributors, and community readers who want to understand where Clawrise is heading.

It focuses on the next set of priorities that matter most, rather than repeating work that has already shipped.  
For completed capabilities, please refer to the project `README` and its current status section.

## 2. What Clawrise Is Focused On Right Now

Clawrise is staying aligned with its core position:

- It is a multi-platform CLI execution layer for agent and automation scenarios.
- It focuses on stable task execution, a consistent calling surface, and a provider model that can keep scaling.
- It is not the dedicated CLI for a single SaaS platform, and it is not trying to become a universal replacement for every existing platform CLI.

For the current OSS phase, the most important decisions are:

- We will prioritize SaaS platforms that have solid APIs, meaningful automation value, and weak or missing CLI ecosystems.
- We will continue pushing the plugin-first direction so platform capabilities live in provider plugins instead of being hard-coded back into the core.
- We will stay open, positive, and respectful toward platforms that already have mature CLIs, but we will not make “rewriting and competing with their full command surface” a near-term goal.

In practice, that means:

- Platforms like Notion, where the API is already useful but CLI support is still clearly lacking, remain strong candidates for focused OSS investment.
- Platforms like Feishu, where a mature CLI already exists, still matter to us and will continue to serve as useful validation samples, but they are not where we plan to compete on command coverage.

## 3. What We Want to Improve First

### 3.1 Make Install, Upgrade, and Verification Easier to Understand

Clawrise has already moved into a plugin-first shape, but the official delivery path for first-party plugins still needs work.

Our next step is to make the following much clearer:

- how first-party plugin releases are packaged and versioned
- how users should install and upgrade them
- how core and plugin compatibility should be understood
- how `plugin verify` should be used in practice

The outcome we want is simple:

- installing a first-party plugin should feel shorter and clearer
- upgrade problems should be visible earlier, before a real execution attempt fails

### 3.2 Shorten the Path to a First Real Call

The command surface is already usable, but the path from a fresh install to one successful real call is still too manual.

We want to keep reducing that first-run friction, especially around:

- a tighter quickstart
- a smoother path through `config init`, `auth check`, `doctor`, and one real execution
- example inputs that match real CLI shapes more closely
- clearer links between playbooks and runnable operations

The goal is not to make users read design documents first.  
The goal is to get them to a real successful call sooner.

### 3.3 Prioritize Platforms Where CLI Support Is Still Missing

For the OSS version of Clawrise, the clearest independent value comes from filling real gaps.

That is why near-term first-party provider work will favor platforms that are:

- already usable through APIs
- clearly valuable for automation
- meaningful in real usage volume or demand
- still underserved by CLI tooling, or lacking a mature community standard

These platforms are the right place to invest because the added value is obvious:

- users are not just getting another wrapper, they are getting a command surface they did not really have before
- contributors can more easily understand why the work matters
- the project can build durable task-level operations, playbooks, and metadata around them

### 3.4 Stay Open and Cooperative with Mature CLIs

We do not want Clawrise to become a closed or adversarial project.

When a platform already has a mature CLI, our default posture should be:

- open
- constructive
- respectful of the existing ecosystem
- focused on complementary value rather than defaulting to direct competition

That does not mean ignoring those platforms.  
It means handling them with more discipline:

- we do not need to rush into cloning full command trees
- we do not need to turn command coverage into a near-term race
- we should not let one platform’s existing ecosystem hijack the project’s priorities

A more realistic direction is:

- keep a small set of high-value first-party capabilities that fit agent and cross-task scenarios
- keep watching for future opportunities to cooperate, interoperate, or complement mature CLIs
- make that position explicit in public docs so users and contributors are not left guessing

### 3.5 Make Plugin Authoring Easier to Follow

If plugin authors have to reverse-engineer large parts of the core, ecosystem growth will stay slow.

So we want to keep improving:

- the plugin author guide
- manifest and compatibility documentation
- local validation and minimal onboarding paths for plugin authors
- clearer guidance around handshake, catalog, and execute testing

The practical goal is to let contributors answer two questions quickly:

- can I build a minimal provider plugin for this platform?
- once I do, how do I know it is compatible with the current core?

### 3.6 Keep `spec`, Completion, and Docs on the Same Source of Truth

Clawrise already has `spec export` and completion. That direction should keep getting tighter, not split into parallel systems.

We want to keep improving:

- a more stable exported metadata contract
- operation reference material generated from the same metadata layer
- a clearer relationship between runtime facts, catalog data, completion, and generated docs

The value here is straightforward:

- users get more consistent capability documentation
- contributors do not need to maintain multiple drifting sources of truth
- agents can consume a more stable discovery surface

## 4. What We Want the Community to Notice

If this phase goes well, the community should see a few visible improvements:

- first-party plugin install, upgrade, and verification paths will be clearer than they are today
- new users will have a shorter path from zero to one real successful call
- the project’s platform selection logic will be easier to understand
- Clawrise’s stance toward mature CLIs will be more explicit and less likely to be misread as “we want to rebuild everything”
- third-party plugin authors will have a more reliable onboarding and compatibility boundary
- `spec`, completion, and operation documentation will feel more unified

## 5. Directions That Still Matter After That

These directions still matter. They just come after the priorities above.

### 5.1 Expand High-signal Playbooks

We want to keep adding playbooks that are closer to real tasks, especially those that are:

- high-frequency
- easy to verify end to end
- good examples of Clawrise’s task-level value

### 5.2 Keep Watching Collaboration Opportunities with Mature CLIs

Feishu will not be the only platform with a mature CLI ecosystem.

Tools like `gh` are worth watching closely as well.  
But that kind of work fits better as a later extension around interoperability and complementary value, not as a higher priority than filling CLI gaps.

### 5.3 Expand First-party Providers Where It Clearly Makes Sense

Expanding provider coverage is still a natural direction, but only when it is justified by real value:

- it should be worth doing
- users should clearly need it
- it should lead to stable task capabilities
- it should not be done just to increase the platform count

## 6. What Is Not a Priority Right Now

These areas are not unimportant. They are just not at the top of the list right now:

- a public plugin marketplace
- sandboxing for untrusted plugins
- a REPL-first interactive shell
- a full JSON Schema framework
- a cross-platform workflow engine
- large provider-specific command tree rewrites done mainly for coverage

## 7. What We Want to Avoid

To keep the project pointed in the right direction, we want to avoid:

- hard-coding platform details back into the core
- turning the project back into a single-platform CLI rewrite effort
- jumping into low-value command coverage competition just because a platform already has a mature ecosystem
- creating multiple drifting metadata sources across `spec`, docs, completion, and runtime
- treating remote install support as if the trust model were already complete
- expanding provider surface too early before we are clear on whether a platform really deserves a first-party provider

## 8. Summary

The next OSS phase for Clawrise is not about trying to do every platform first.  
It is about doing the most valuable parts well enough that users and contributors can clearly see why the project matters.

In the near term, that means:

- making the plugin-first path more solid
- shortening the first-run experience
- going deeper on platforms where CLI support is still missing
- making our stance toward mature CLIs more explicit
- making community participation easier around extension and integration paths

If we do that well, Clawrise should become easier to understand, easier to adopt, and easier to contribute to.

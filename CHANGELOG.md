# [1.1.0](https://github.com/farlleyferreira/conjecture-ai/compare/v1.0.1...v1.1.0) (2026-11-03)


### Features

* bootstrap OpenSpec capability baseline for conjecture ([#15](https://github.com/farlleyferreira/conjecture-ai/issues/15)) ([fdc440d](https://github.com/farlleyferreira/conjecture-ai/commit/fdc440d4d33e43968e4aeea33b23fd7c8f013951))

## [1.2.0](https://github.com/farlleyferreira/conjecture-ai/compare/v1.1.0...v1.2.0) (2026-11-13)


### Features

* add .cjtr composite identifier helpers ([#19](https://github.com/farlleyferreira/conjecture-ai/issues/19)) ([e53cf80](https://github.com/farlleyferreira/conjecture-ai/commit/e53cf8074fc44fbb9d3454761d3057119f44df5d))
* add .cjtr format versioning (conjecture: MAJOR.MINOR.PATCH) ([#18](https://github.com/farlleyferreira/conjecture-ai/issues/18)) ([a553b1d](https://github.com/farlleyferreira/conjecture-ai/commit/a553b1dcaaecb12f72edebf7a2b7e7fc82ba6e96))
* add "conjecture status" and expose it over MCP ([#152](https://github.com/farlleyferreira/conjecture-ai/issues/152)) ([fbd4346](https://github.com/farlleyferreira/conjecture-ai/commit/fbd4346d240a085acbfaeb6cc7d6c9773952691d))
* add "conjecture verify" - the first end-to-end command ([#27](https://github.com/farlleyferreira/conjecture-ai/issues/27)) ([e9f178d](https://github.com/farlleyferreira/conjecture-ai/commit/e9f178d31a818add8ac16c936e88997bda030e93))
* add conjecture base add - the first end-to-end knowledge-base command ([#51](https://github.com/farlleyferreira/conjecture-ai/issues/51)) ([0a508ef](https://github.com/farlleyferreira/conjecture-ai/commit/0a508eff09fe631d1e2a4940584e11023aa31dec))
* add conjecture sign - the first end-to-end signing command ([#49](https://github.com/farlleyferreira/conjecture-ai/issues/49)) ([fddcb22](https://github.com/farlleyferreira/conjecture-ai/commit/fddcb22a9b2533a913a5c8edc1e896fb5bbecf8b))
* add the closed .cjtr vocabulary as a single data source ([#23](https://github.com/farlleyferreira/conjecture-ai/issues/23)) ([5f16109](https://github.com/farlleyferreira/conjecture-ai/commit/5f16109065c23c0660961eb527e4e83322fa0696))
* add yamlstrict, a strict-profile YAML loader ([#17](https://github.com/farlleyferreira/conjecture-ai/issues/17)) ([fa77b1f](https://github.com/farlleyferreira/conjecture-ai/commit/fa77b1fbbda2feb82dc864ead9afb10bd20476f2))
* **adversarial-slm:** add SLM selection, output grammar, blind input ([#63](https://github.com/farlleyferreira/conjecture-ai/issues/63)) ([b7ef910](https://github.com/farlleyferreira/conjecture-ai/commit/b7ef910f49ee536a880d430b9791a52b07191549))
* **adversarial-slm:** add the Ollama-backed local runner ([#73](https://github.com/farlleyferreira/conjecture-ai/issues/73)) ([c9fc07f](https://github.com/farlleyferreira/conjecture-ai/commit/c9fc07fcd6f8144b19cca4e63f99042aad819063))
* **adversarial-slm:** wire a real local attack into the CLI ([#130](https://github.com/farlleyferreira/conjecture-ai/issues/130)) ([b03ba56](https://github.com/farlleyferreira/conjecture-ai/commit/b03ba56b24d56cd14a9c30ced6291a124697f7f4))
* **attacker-swarm:** add mode validation, bandit allocator, aggregation ([#62](https://github.com/farlleyferreira/conjecture-ai/issues/62)) ([1408e49](https://github.com/farlleyferreira/conjecture-ai/commit/1408e4917c4b9bfc64fb2852ca9c882bc86c32b5))
* **attacker-swarm:** wire real multi-agent dispatch into the CLI ([#136](https://github.com/farlleyferreira/conjecture-ai/issues/136)) ([8e8caa4](https://github.com/farlleyferreira/conjecture-ai/commit/8e8caa419be55679e2a2e4027c159f803772bca9))
* **attack:** warn when attacking a law with a single local model ([#158](https://github.com/farlleyferreira/conjecture-ai/issues/158)) ([b164e4e](https://github.com/farlleyferreira/conjecture-ai/commit/b164e4e95fa5d18ded9f92d185327643db370133))
* **bootstrap-material:** add known-answer probe checking ([#57](https://github.com/farlleyferreira/conjecture-ai/issues/57)) ([57449c5](https://github.com/farlleyferreira/conjecture-ai/commit/57449c5206d24718edee5ec84ea1a2f55d88cd58))
* check user_defined_laws at the phenomenon level too ([#35](https://github.com/farlleyferreira/conjecture-ai/issues/35)) ([8613b55](https://github.com/farlleyferreira/conjecture-ai/commit/8613b55f36b11d6efa25e269f7ddc275153a0418))
* **cjtr:** generate deterministic "# pt:" comments from the vocabulary ([#78](https://github.com/farlleyferreira/conjecture-ai/issues/78)) ([298c6d5](https://github.com/farlleyferreira/conjecture-ai/commit/298c6d5f248d254673c13d0cc4a7b7f59531a8cc))
* **cjtr:** wire Evaluate's comparison step through expression-core ([#100](https://github.com/farlleyferreira/conjecture-ai/issues/100)) ([209c33c](https://github.com/farlleyferreira/conjecture-ai/commit/209c33c1dc2b3204f3c96da74dc3105421cb35ab))
* **cmd:** accept JUnit/TAP results as conformance evidence ([#92](https://github.com/farlleyferreira/conjecture-ai/issues/92)) ([c87efdb](https://github.com/farlleyferreira/conjecture-ai/commit/c87efdb93f9af30d15be57fe0f9e3d140a7919b8))
* **cmd:** add --scope to "conjecture verify" ([#113](https://github.com/farlleyferreira/conjecture-ai/issues/113)) ([104ca36](https://github.com/farlleyferreira/conjecture-ai/commit/104ca369b605a0a6d64d1bf88499cf995ee1bd6a))
* **cmd:** add "conjecture context resolve-conflict" ([#112](https://github.com/farlleyferreira/conjecture-ai/issues/112)) ([1b9baab](https://github.com/farlleyferreira/conjecture-ai/commit/1b9baab4240a9436c4979e7cd40dc79f76e63ef4))
* **cmd:** add "conjecture law resolve-conflict" ([#111](https://github.com/farlleyferreira/conjecture-ai/issues/111)) ([5510e73](https://github.com/farlleyferreira/conjecture-ai/commit/5510e732c81e53e4accb4de9361221acb6589446))
* **cmd:** add "conjecture optimize" calibration gate ([#117](https://github.com/farlleyferreira/conjecture-ai/issues/117)) ([0cde2bf](https://github.com/farlleyferreira/conjecture-ai/commit/0cde2bf12115b87b09f5738fd78ac051bac462ae))
* **cmd:** add "conjecture settings check-floor" ([#115](https://github.com/farlleyferreira/conjecture-ai/issues/115)) ([67d10c0](https://github.com/farlleyferreira/conjecture-ai/commit/67d10c0c81f121d70d1d7cdb4aebfe0140451450))
* **cmd:** add "conjecture translate" ([#114](https://github.com/farlleyferreira/conjecture-ai/issues/114)) ([271ce6d](https://github.com/farlleyferreira/conjecture-ai/commit/271ce6db832d77f13a98dd8a17e3532795a40e0e))
* **cmd:** add "conjecture watch", the folder-watch channel ([#123](https://github.com/farlleyferreira/conjecture-ai/issues/123)) ([28e129f](https://github.com/farlleyferreira/conjecture-ai/commit/28e129ffc6243c1a3b60f72070e37139aafa2966))
* **cmd:** add "conjecture work", the work-queue's own orchestrator loop ([#105](https://github.com/farlleyferreira/conjecture-ai/issues/105)) ([f5c9e5b](https://github.com/farlleyferreira/conjecture-ai/commit/f5c9e5b211201c7c62f697aecbe22209dd74529b))
* **cmd:** add a "read_evidence" MCP tool ([#108](https://github.com/farlleyferreira/conjecture-ai/issues/108)) ([74deb6d](https://github.com/farlleyferreira/conjecture-ai/commit/74deb6d42024c8d9781403f008469eb250d01c64))
* **cmd:** add an MCP server exposing a "verify" tool ([#104](https://github.com/farlleyferreira/conjecture-ai/issues/104)) ([0692ca1](https://github.com/farlleyferreira/conjecture-ai/commit/0692ca1474a32b6771b169c6f3739b997c24a49b))
* **cmd:** add conjecture check orchestrator loop ([#82](https://github.com/farlleyferreira/conjecture-ai/issues/82)) ([e2ffb59](https://github.com/farlleyferreira/conjecture-ai/commit/e2ffb593a5dd826517df2f5b10ffb56ef0ffa44f))
* **cmd:** add conjecture implement round loop ([#83](https://github.com/farlleyferreira/conjecture-ai/issues/83)) ([06639e3](https://github.com/farlleyferreira/conjecture-ai/commit/06639e33c16825748266e5a301942ac51a32bc56))
* **cmd:** context resume reports the next work-queue item ([#90](https://github.com/farlleyferreira/conjecture-ai/issues/90)) ([01d69e4](https://github.com/farlleyferreira/conjecture-ai/commit/01d69e4a885fedd174919c81ac78c69b0e2574b3))
* **cmd:** enqueue a blocker work-queue item on violation ([#89](https://github.com/farlleyferreira/conjecture-ai/issues/89)) ([2a0808a](https://github.com/farlleyferreira/conjecture-ai/commit/2a0808a849f1014f20644dea7022dba43a394fce))
* **cmd:** escalate a repeatedly-violated law's status to contested ([#93](https://github.com/farlleyferreira/conjecture-ai/issues/93)) ([b38fff6](https://github.com/farlleyferreira/conjecture-ai/commit/b38fff62a4b82960d36cde14499f6a9db59bbcaa))
* **cmd:** mark settings.cjtr unconfirmed on a rigor-reducing calibration ([#125](https://github.com/farlleyferreira/conjecture-ai/issues/125)) ([ac0ea7b](https://github.com/farlleyferreira/conjecture-ai/commit/ac0ea7b02fd277f9ba6d9e87cc87c85b8f386711))
* **cmd:** promote a law to Thesis on enough failure-free cases ([#94](https://github.com/farlleyferreira/conjecture-ai/issues/94)) ([01402e9](https://github.com/farlleyferreira/conjecture-ai/commit/01402e9899ba7a6517e26f474e0671ee5b025b9b))
* **cmd:** scope "conjecture base search" to an area ([#95](https://github.com/farlleyferreira/conjecture-ai/issues/95)) ([7a8a845](https://github.com/farlleyferreira/conjecture-ai/commit/7a8a845cf725353f4e1441caca651c3abcefc36c))
* **cmd:** warn when prior evidence is stale against this run ([#96](https://github.com/farlleyferreira/conjecture-ai/issues/96)) ([151d7c4](https://github.com/farlleyferreira/conjecture-ai/commit/151d7c43aca7797595b3f14d55601d3b9fb4ae0d))
* **cmd:** wire omission-detection's Type D check into "conjecture verify" ([#110](https://github.com/farlleyferreira/conjecture-ai/issues/110)) ([355810a](https://github.com/farlleyferreira/conjecture-ai/commit/355810a918d7b6978c6b55c80572c71781c0183f))
* **cmd:** wire spec-judge's consistency check via solver into verify ([#103](https://github.com/farlleyferreira/conjecture-ai/issues/103)) ([f14b613](https://github.com/farlleyferreira/conjecture-ai/commit/f14b613002d45faf9cdf0362427b0ad26b6141d0))
* **cmd:** wire the user-defined-law status lifecycle into check/implement ([#91](https://github.com/farlleyferreira/conjecture-ai/issues/91)) ([ede478d](https://github.com/farlleyferreira/conjecture-ai/commit/ede478d991878496f223a3e4f88e54e21f0a62d5))
* **conformance-cycle:** add budget, counterexample, and rigor rules ([#75](https://github.com/farlleyferreira/conjecture-ai/issues/75)) ([585edae](https://github.com/farlleyferreira/conjecture-ai/commit/585edae0b3248de294df3d2d4a8de0fda86103f9))
* **conformance-cycle:** add the no-network execution sandbox ([#74](https://github.com/farlleyferreira/conjecture-ai/issues/74)) ([2afbde9](https://github.com/farlleyferreira/conjecture-ai/commit/2afbde9dd85347b1d10707d15e7e9f943faf9985))
* **conformance-cycle:** add the three-way verdict type ([#55](https://github.com/farlleyferreira/conjecture-ai/issues/55)) ([13f7f01](https://github.com/farlleyferreira/conjecture-ai/commit/13f7f0156bdcb2184dd58e6956756ad238083afe))
* **conformance-cycle:** connect the sandbox to a real hypothesis check ([#76](https://github.com/farlleyferreira/conjecture-ai/issues/76)) ([121fe65](https://github.com/farlleyferreira/conjecture-ai/commit/121fe654a4b71178f0c77e6c3fa2935cd8f7983c))
* **conformance-cycle:** generate boundary values for a concept domain ([#39](https://github.com/farlleyferreira/conjecture-ai/issues/39)) ([85bd63f](https://github.com/farlleyferreira/conjecture-ai/commit/85bd63fd883764af70a2f60ea1ecd985698bfdb2))
* **conformance-cycle:** run directed sampling for real against implementation ([#80](https://github.com/farlleyferreira/conjecture-ai/issues/80)) ([64e1406](https://github.com/farlleyferreira/conjecture-ai/commit/64e1406c0b9ff64a908d524cb9c9c02b8f619921))
* **conformancecycle:** implement bounded and complete proof rigor levels ([#107](https://github.com/farlleyferreira/conjecture-ai/issues/107)) ([267a82c](https://github.com/farlleyferreira/conjecture-ai/commit/267a82c6d62261e0e78a9bb71793be664e076665))
* **conformancecycle:** wire the evolutionary input generator ([#118](https://github.com/farlleyferreira/conjecture-ai/issues/118)) ([c8f599a](https://github.com/farlleyferreira/conjecture-ai/commit/c8f599a96e5f5f24801714cc10c320a8f51542f1))
* **context-composition:** add conflict precedence by mode ([#67](https://github.com/farlleyferreira/conjecture-ai/issues/67)) ([4ac3538](https://github.com/farlleyferreira/conjecture-ai/commit/4ac35385b4df6476707e6b2141128d2969bdc979))
* **context-composition:** add context/version storage ([#69](https://github.com/farlleyferreira/conjecture-ai/issues/69)) ([3e43d93](https://github.com/farlleyferreira/conjecture-ai/commit/3e43d93720cfdbe7f9a9e3fb261d5a73f1a727b6))
* **context-composition:** add lifecycle state and composition modes ([#66](https://github.com/farlleyferreira/conjecture-ai/issues/66)) ([a9ce26e](https://github.com/farlleyferreira/conjecture-ai/commit/a9ce26e317ee308f172811bf1a1a1768ee72f613))
* **context-composition:** add the CLI for list/resume/open/restore ([#70](https://github.com/farlleyferreira/conjecture-ai/issues/70)) ([82d3e1e](https://github.com/farlleyferreira/conjecture-ai/commit/82d3e1e40a40d03a862dc447c38b441260d257c9))
* **context-composition:** filter conjecture context list by area and date ([#139](https://github.com/farlleyferreira/conjecture-ai/issues/139)) ([a6bfea5](https://github.com/farlleyferreira/conjecture-ai/commit/a6bfea51ed9805cffa4af7e9c5f905b1e5e4677f))
* **context-composition:** implement composition by reference for real ([#81](https://github.com/farlleyferreira/conjecture-ai/issues/81)) ([b82d8e4](https://github.com/farlleyferreira/conjecture-ai/commit/b82d8e43e0e2d12f1e751b7c5a842858163d6bb5))
* **context-engine:** add weight-gating and relevance propagation ([#61](https://github.com/farlleyferreira/conjecture-ai/issues/61)) ([c03ddbc](https://github.com/farlleyferreira/conjecture-ai/commit/c03ddbc384b60f6f8b3153009ceb1d00cc91e79d))
* **context-engine:** persist per-node weight/relevance/confidence ([#124](https://github.com/farlleyferreira/conjecture-ai/issues/124)) ([6db9228](https://github.com/farlleyferreira/conjecture-ai/commit/6db9228842c874500ba6d18229ad0e8b4b264e5a))
* **context:** wire conjecture context origin and diff ([#147](https://github.com/farlleyferreira/conjecture-ai/issues/147)) ([7018527](https://github.com/farlleyferreira/conjecture-ai/commit/70185275d0f0ce4c05763b8773073c70cb44c9ba))
* decision ledger and conjecture init (persistence-decisions-and-layout) ([#145](https://github.com/farlleyferreira/conjecture-ai/issues/145)) ([29321aa](https://github.com/farlleyferreira/conjecture-ai/commit/29321aa2c4d52eb7047ec703bfcf730ccea434b6))
* evaluate .cjtr comparison statements against real bindings ([#25](https://github.com/farlleyferreira/conjecture-ai/issues/25)) ([fa0840e](https://github.com/farlleyferreira/conjecture-ai/commit/fa0840ed62c57043d64c4ac19f6dfd3409384f4b))
* **evidence-ledger:** implement the maturity scale decision rules ([#44](https://github.com/farlleyferreira/conjecture-ai/issues/44)) ([021c36f](https://github.com/farlleyferreira/conjecture-ai/commit/021c36f771d69ddb33a2eb34ca28c04e93a140c2))
* **evidence-ledger:** wire single-counterexample demotion ([#85](https://github.com/farlleyferreira/conjecture-ai/issues/85)) ([50db1c1](https://github.com/farlleyferreira/conjecture-ai/commit/50db1c1f669d564c9108f56a3ee8425dd4e6e34f))
* **evidence-ledger:** wire the maturity scale into the SQLite store ([#45](https://github.com/farlleyferreira/conjecture-ai/issues/45)) ([c26ae8c](https://github.com/farlleyferreira/conjecture-ai/commit/c26ae8c7f3e8d66c782263ec96f8da959b303d9b))
* **evidence-ledger:** wire Thesis-&gt;Theory and Theory-&gt;Law promotion ([#135](https://github.com/farlleyferreira/conjecture-ai/issues/135)) ([9deb5ab](https://github.com/farlleyferreira/conjecture-ai/commit/9deb5abc61d959318746fc02d05a3e48f3cb2046))
* **expression-core:** compile a cjtr comparator to a CEL program ([#99](https://github.com/farlleyferreira/conjecture-ai/issues/99)) ([18a477e](https://github.com/farlleyferreira/conjecture-ai/commit/18a477e42ad7083fe97b2efd63130afc060a1d1b))
* **expression-core:** exact-decimal CEL type via cel-go ([#97](https://github.com/farlleyferreira/conjecture-ai/issues/97)) ([5073223](https://github.com/farlleyferreira/conjecture-ai/commit/5073223c98ecbbfaba249fe2e12c5a080160fc46))
* **expression-core:** invoke z3 as a subprocess to check satisfiability ([#102](https://github.com/farlleyferreira/conjecture-ai/issues/102)) ([d52a79b](https://github.com/farlleyferreira/conjecture-ai/commit/d52a79bd7d65c0a6ba35c64429b5b84a491c3e3e))
* **expression-core:** translate a comparator to an SMT-LIB2 constraint ([#101](https://github.com/farlleyferreira/conjecture-ai/issues/101)) ([4503e2d](https://github.com/farlleyferreira/conjecture-ai/commit/4503e2d03f46202f18e7949d5605dd1462e6d7ec))
* **hardware-scheduler:** add the defer-heavy-work policy ([#84](https://github.com/farlleyferreira/conjecture-ai/issues/84)) ([2b587ea](https://github.com/farlleyferreira/conjecture-ai/commit/2b587ea24f9d0725d3c580972bd7c35642228d1a))
* **hardware-scheduler:** add the minimum RAM check ([#54](https://github.com/farlleyferreira/conjecture-ai/issues/54)) ([4423a4c](https://github.com/farlleyferreira/conjecture-ai/commit/4423a4c5425dc8c3bc678e98a7595b1025d60da7))
* **hardware-scheduler:** measure CPU load and wire it into watch ([#128](https://github.com/farlleyferreira/conjecture-ai/issues/128)) ([62c7ce1](https://github.com/farlleyferreira/conjecture-ai/commit/62c7ce1d3ea71570d75e7b41d95f96aabcb46478))
* **implement:** refuse the disposable oracle on an unbounded domain ([#155](https://github.com/farlleyferreira/conjecture-ai/issues/155)) ([a43a0bc](https://github.com/farlleyferreira/conjecture-ai/commit/a43a0bc0bfa44fe3c6e9ef3f41e10bb3421d7f59))
* **init:** update the scaffolded skill when it is safe to ([#163](https://github.com/farlleyferreira/conjecture-ai/issues/163)) ([f84a60a](https://github.com/farlleyferreira/conjecture-ai/commit/f84a60afb5688afa01f678bb9f402834ee1758f1))
* **knowledge-bases:** add local full-text search ([#86](https://github.com/farlleyferreira/conjecture-ai/issues/86)) ([4071713](https://github.com/farlleyferreira/conjecture-ai/commit/4071713c35132e0d36678f6057f6515689624b53))
* **knowledge-bases:** add the authority model and registration store ([#50](https://github.com/farlleyferreira/conjecture-ai/issues/50)) ([01ad9db](https://github.com/farlleyferreira/conjecture-ai/commit/01ad9db3af627bea40a151610221e37bcb5710c1))
* **knowledge-bases:** implement quantized local embeddings ([#129](https://github.com/farlleyferreira/conjecture-ai/issues/129)) ([cd106b7](https://github.com/farlleyferreira/conjecture-ai/commit/cd106b73a36b6cf6776f8eb44cfddce18830478f))
* **language-agnostic-contracts:** add a JUnit XML result parser ([#40](https://github.com/farlleyferreira/conjecture-ai/issues/40)) ([0010c67](https://github.com/farlleyferreira/conjecture-ai/commit/0010c67ed24582ad4cd319ff98419d90238ee410))
* **language-agnostic-contracts:** add a TAP result parser ([#41](https://github.com/farlleyferreira/conjecture-ai/issues/41)) ([9a3e681](https://github.com/farlleyferreira/conjecture-ai/commit/9a3e6811f73a64eeebada6983d2e4f632e5a021b))
* **language-agnostic-contracts:** add JSON Schema validation ([#42](https://github.com/farlleyferreira/conjecture-ai/issues/42)) ([d52786c](https://github.com/farlleyferreira/conjecture-ai/commit/d52786cf8007b63fe6970707b3f89e343df7d32c))
* **language-agnostic-contracts:** define the GUI accessibility driver contract ([#132](https://github.com/farlleyferreira/conjecture-ai/issues/132)) ([9ea102a](https://github.com/farlleyferreira/conjecture-ai/commit/9ea102a170b95d3cd94373789fb61d7bc7f4d977))
* **language-agnostic-contracts:** implement read-only dependency analysis ([#131](https://github.com/farlleyferreira/conjecture-ai/issues/131)) ([0504df1](https://github.com/farlleyferreira/conjecture-ai/commit/0504df197b23087cd1a8f8eab9ed0f2bef3c4639))
* **languageagnosticcontracts:** real Windows and Linux GUI driver backends ([#142](https://github.com/farlleyferreira/conjecture-ai/issues/142)) ([8c9c818](https://github.com/farlleyferreira/conjecture-ai/commit/8c9c8181e4af9737d3834e168f4ec323b4579143))
* **llm-orchestration:** add a provider-agnostic real LLM client ([#71](https://github.com/farlleyferreira/conjecture-ai/issues/71)) ([bad9450](https://github.com/farlleyferreira/conjecture-ai/commit/bad9450c86aa82c6e7e83b9751d690ea39c89e03))
* **llm-orchestration:** add session separation and trust-boundary gate ([#64](https://github.com/farlleyferreira/conjecture-ai/issues/64)) ([3a99711](https://github.com/farlleyferreira/conjecture-ai/commit/3a997115b235c710eb270ee30624d5859553e1c7))
* **llm-orchestration:** add the conjecture draft command ([#72](https://github.com/farlleyferreira/conjecture-ai/issues/72)) ([08d2324](https://github.com/farlleyferreira/conjecture-ai/commit/08d2324d1f2bce1da06cdd128f6a0252b856b36f))
* **llmclient:** add 9 OpenAI-compatible provider presets ([#167](https://github.com/farlleyferreira/conjecture-ai/issues/167)) ([dad5693](https://github.com/farlleyferreira/conjecture-ai/commit/dad56932589a74de51e4980dcf1b04ba9ebdfd61))
* **llmclient:** add a local llama.cpp provider ([#140](https://github.com/farlleyferreira/conjecture-ai/issues/140)) ([e4b3834](https://github.com/farlleyferreira/conjecture-ai/commit/e4b3834929adef4fed7c4e82c99b9a62b2657bd8))
* **llmclient:** add DeepSeek provider implementation ([#77](https://github.com/farlleyferreira/conjecture-ai/issues/77)) ([1d3866c](https://github.com/farlleyferreira/conjecture-ai/commit/1d3866c2f0716f4f67ae98201351979262f953fa))
* **mcp:** expose conformance checking as an MCP tool ([#150](https://github.com/farlleyferreira/conjecture-ai/issues/150)) ([a0288c3](https://github.com/farlleyferreira/conjecture-ai/commit/a0288c35ed1ee8a4b6486c95e532081661aff192))
* **mcp:** expose every other read-only CLI command as a tool ([#153](https://github.com/farlleyferreira/conjecture-ai/issues/153)) ([b90a49e](https://github.com/farlleyferreira/conjecture-ai/commit/b90a49e356f5dedbafc1e7427fb3bfa683df62db))
* **omission-detection:** add the Type D catalog gap check ([#59](https://github.com/farlleyferreira/conjecture-ai/issues/59)) ([b25c4c8](https://github.com/farlleyferreira/conjecture-ai/commit/b25c4c81a4e8bbf95f4079b391b18c1cb50c3c47))
* **omission-detection:** detect Type A omissions for real ([#87](https://github.com/farlleyferreira/conjecture-ai/issues/87)) ([61642bb](https://github.com/farlleyferreira/conjecture-ai/commit/61642bbd6479a4e061e89da70503af35b0854afa))
* **omission-detection:** implement Type B - case forgotten within the domain ([#126](https://github.com/farlleyferreira/conjecture-ai/issues/126)) ([b83fe63](https://github.com/farlleyferreira/conjecture-ai/commit/b83fe631a54479c1747cdddabf42c54bfd0a91d3))
* **omission-detection:** implement Type C's decision-table half ([#127](https://github.com/farlleyferreira/conjecture-ai/issues/127)) ([d7ec513](https://github.com/farlleyferreira/conjecture-ai/commit/d7ec5130c2bb2639e6f1a411fb9fa1b8342330bf))
* **optimization-suite:** add calibration approval gate and floor clamp ([#65](https://github.com/farlleyferreira/conjecture-ai/issues/65)) ([3fe7e30](https://github.com/farlleyferreira/conjecture-ai/commit/3fe7e30976fb6d191c5512966c0eb1d3f72cef1f))
* **optimization-suite:** add the evolutionary input generator ([#68](https://github.com/farlleyferreira/conjecture-ai/issues/68)) ([b67cb78](https://github.com/farlleyferreira/conjecture-ai/commit/b67cb783a1d69b5615caa88a71ccca674f878295))
* parse .cjtr comparison statements (given/then/otherwise/always) ([#24](https://github.com/farlleyferreira/conjecture-ai/issues/24)) ([ff812be](https://github.com/farlleyferreira/conjecture-ai/commit/ff812be829e8cd9bc22234083f44d40ab07fc5ef))
* parse .cjtr concepts (typed variables and constants) ([#22](https://github.com/farlleyferreira/conjecture-ai/issues/22)) ([b331c98](https://github.com/farlleyferreira/conjecture-ai/commit/b331c983abfa97ec53fd9eb8d9e1d0074d680a40))
* parse .cjtr hypotheses and check examples for consistency ([#26](https://github.com/farlleyferreira/conjecture-ai/issues/26)) ([654875a](https://github.com/farlleyferreira/conjecture-ai/commit/654875a1204032c6334162e2032a9f85b83ee7f1))
* parse .cjtr rules and compose their IDs from real document shape ([#20](https://github.com/farlleyferreira/conjecture-ai/issues/20)) ([a568fab](https://github.com/farlleyferreira/conjecture-ai/commit/a568fabca34f36cee820096eea3c5be2b7fb4ce0))
* parse a file top-level scope field ([#36](https://github.com/farlleyferreira/conjecture-ai/issues/36)) ([f515b97](https://github.com/farlleyferreira/conjecture-ai/commit/f515b97380e226f82b3b0e1deb19bfc34ae99b10))
* parse a file top-level simplifications field ([#37](https://github.com/farlleyferreira/conjecture-ai/issues/37)) ([b090917](https://github.com/farlleyferreira/conjecture-ai/commit/b0909170e407fbca6239af86f8debe4577d40c98))
* parse and verify a rule user_defined_laws ([#33](https://github.com/farlleyferreira/conjecture-ai/issues/33)) ([ee5e555](https://github.com/farlleyferreira/conjecture-ai/commit/ee5e5557bb92d1c165edd0d0e92259b486088ee5))
* parse observation and validate a rule when against it ([#34](https://github.com/farlleyferreira/conjecture-ai/issues/34)) ([4da41b3](https://github.com/farlleyferreira/conjecture-ai/commit/4da41b3776e741b29f9897f623ccaeed3c2dea43))
* parse the axioms assumes clause ([#38](https://github.com/farlleyferreira/conjecture-ai/issues/38)) ([45846c6](https://github.com/farlleyferreira/conjecture-ai/commit/45846c67dfc514c932d6fbf989c199b8f5ab0c9b))
* parse the decisions hypothesis clause ([#32](https://github.com/farlleyferreira/conjecture-ai/issues/32)) ([635b84f](https://github.com/farlleyferreira/conjecture-ai/commit/635b84fbae16e82dc63dcb3aba24eda0f3795bb5))
* parse the unchanged hypothesis clause ([#30](https://github.com/farlleyferreira/conjecture-ai/issues/30)) ([17990c9](https://github.com/farlleyferreira/conjecture-ai/commit/17990c9c449fce8d6ac9cf2ca64e8372d15b229b))
* **persistence:** add the per-repository SQLite local store ([#43](https://github.com/farlleyferreira/conjecture-ai/issues/43)) ([8c67437](https://github.com/farlleyferreira/conjecture-ai/commit/8c6743713a3341c0e2abf26c3198adffed8c6861))
* **queue:** group "queue list" by area::phenomenon, bound its scale ([#157](https://github.com/farlleyferreira/conjecture-ai/issues/157)) ([d4c9cfd](https://github.com/farlleyferreira/conjecture-ai/commit/d4c9cfdc376319fd2004425d9a8bafa737f45aae))
* **queue:** warn when a drill-down is 3x the project's own average ([#160](https://github.com/farlleyferreira/conjecture-ai/issues/160)) ([1cf94bb](https://github.com/farlleyferreira/conjecture-ai/commit/1cf94bb150d2bd3f6ae2c254a153c3feff4b857e))
* resolve table-lookup concepts against CSV files ([#28](https://github.com/farlleyferreira/conjecture-ai/issues/28)) ([668b31d](https://github.com/farlleyferreira/conjecture-ai/commit/668b31d312e982ec025d3ab69087297d388c26d3))
* scaffold a conjecture-workflow skill from "conjecture init" ([#154](https://github.com/farlleyferreira/conjecture-ai/issues/154)) ([e44cb53](https://github.com/farlleyferreira/conjecture-ai/commit/e44cb53ae62f1f7355094b551a3a24142eb07588))
* scaffold Go module and CLI entrypoint ([#16](https://github.com/farlleyferreira/conjecture-ai/issues/16)) ([5cdb9b5](https://github.com/farlleyferreira/conjecture-ai/commit/5cdb9b5c8c80dfd69d3f09accf663ea4c4ea8a3f))
* **spec-authoring:** add the cjtr signature mechanism ([#47](https://github.com/farlleyferreira/conjecture-ai/issues/47)) ([9e15107](https://github.com/farlleyferreira/conjecture-ai/commit/9e15107b42b133f335ff0da2ee1206d8ccd1e69f))
* **spec-authoring:** detect pending markers that block signing ([#48](https://github.com/farlleyferreira/conjecture-ai/issues/48)) ([76e4283](https://github.com/farlleyferreira/conjecture-ai/commit/76e428367584ca903c5ee2378af7a1c5eff8ce71))
* **spec-judge:** add the accepted pending rejected verdict aggregator ([#56](https://github.com/farlleyferreira/conjecture-ai/issues/56)) ([d12b5b1](https://github.com/farlleyferreira/conjecture-ai/commit/d12b5b16f64c722357814dd98986dc80655354ba))
* **spec-judge:** add the falsifiability check and wire it into verify ([#53](https://github.com/farlleyferreira/conjecture-ai/issues/53)) ([28428da](https://github.com/farlleyferreira/conjecture-ai/commit/28428da5355cf1fde7e885dfe47802cb55de2eae))
* **status:** add a project-wide aggregate ([#159](https://github.com/farlleyferreira/conjecture-ai/issues/159)) ([71fa5d2](https://github.com/farlleyferreira/conjecture-ai/commit/71fa5d2ea856c6a38de03ec3cf090a9c741f21e5))
* support the is-between/is-strictly-between comparators ([#29](https://github.com/farlleyferreira/conjecture-ai/issues/29)) ([cb267fa](https://github.com/farlleyferreira/conjecture-ai/commit/cb267fa935f84007f07992cc4c590c8ab37a7ffe))
* **swarm:** optional per-strategy base-url for distinct local model families ([#141](https://github.com/farlleyferreira/conjecture-ai/issues/141)) ([ac70575](https://github.com/farlleyferreira/conjecture-ai/commit/ac705754d131b126edf196768f48889b915177ca))
* **team-config:** add the generic floor-not-ceiling comparison ([#46](https://github.com/farlleyferreira/conjecture-ai/issues/46)) ([b1c4879](https://github.com/farlleyferreira/conjecture-ai/commit/b1c487967462e63a0ec72943bf14e9928eb23185))
* **teamconfig:** add the settings.cjtr schema and loader ([#116](https://github.com/farlleyferreira/conjecture-ai/issues/116)) ([1d9a53a](https://github.com/farlleyferreira/conjecture-ai/commit/1d9a53a2bf6cdb2115c1e210f59c48be8ee1e1f3))
* **user-defined-laws:** status lifecycle and invalidation pathway ([#79](https://github.com/farlleyferreira/conjecture-ai/issues/79)) ([1ca7280](https://github.com/farlleyferreira/conjecture-ai/commit/1ca7280999203684282f9709091f0575e40de01c))
* **work-queue:** implement the single three-tier priority queue ([#52](https://github.com/farlleyferreira/conjecture-ai/issues/52)) ([f8e4f08](https://github.com/farlleyferreira/conjecture-ai/commit/f8e4f0801a4c842b107fa9e4827d015fc86ea008))
* **work-queue:** persist the queue and expose it via the CLI ([#88](https://github.com/farlleyferreira/conjecture-ai/issues/88)) ([e7ee70e](https://github.com/farlleyferreira/conjecture-ai/commit/e7ee70e3356d43c4d421b17bf7ac1873d9ce281a))
* **work:** dead-letter a refused item, halt on an unresolved blocker ([#164](https://github.com/farlleyferreira/conjecture-ai/issues/164)) ([86f234f](https://github.com/farlleyferreira/conjecture-ai/commit/86f234f2e54572a81c7b8805ad3b92115f85e45b))
* **workqueue:** add dead-letter resolve ([#165](https://github.com/farlleyferreira/conjecture-ai/issues/165)) ([3f465d6](https://github.com/farlleyferreira/conjecture-ai/commit/3f465d69d81f809789878cc6af45405b66ded11a))


### Bug Fixes

* **ci:** bump Release workflow's Node version to unbreak semantic-release ([#172](https://github.com/farlleyferreira/conjecture-ai/issues/172)) ([4e9588d](https://github.com/farlleyferreira/conjecture-ai/commit/4e9588df34b0aee34f1d1f4e7225a4ab321e91c0))
* **ci:** replace semantic-release with release-please (PR-based) ([#173](https://github.com/farlleyferreira/conjecture-ai/issues/173)) ([4bedd68](https://github.com/farlleyferreira/conjecture-ai/commit/4bedd682b38b38192329ef8e19208368853160aa))
* **ci:** trigger the PR-title check on pull_request, not pull_request_target ([#175](https://github.com/farlleyferreira/conjecture-ai/issues/175)) ([85dd087](https://github.com/farlleyferreira/conjecture-ai/commit/85dd08709c71a446415c5a93a2ec0b85fe55f23d))
* **cmd:** correct mcp.go's stale doc comment on closed gaps ([#134](https://github.com/farlleyferreira/conjecture-ai/issues/134)) ([eb0f3f0](https://github.com/farlleyferreira/conjecture-ai/commit/eb0f3f0c564e52b6bca1fcf83567cef259937cf6))
* **cmd:** correct stale doc comments on closed gaps ([#137](https://github.com/farlleyferreira/conjecture-ai/issues/137)) ([d3f3b08](https://github.com/farlleyferreira/conjecture-ai/commit/d3f3b088cab067306c9d15bd382de4b4e9359d19))
* **conformancecycle:** stop leaking the expected output into the sandboxed input file ([#143](https://github.com/farlleyferreira/conjecture-ai/issues/143)) ([e2504ba](https://github.com/farlleyferreira/conjecture-ai/commit/e2504baccda4c74f12c108b537009f4087c2d246))
* **draft:** share the cjtr grammar reference with the scaffolded skill ([#161](https://github.com/farlleyferreira/conjecture-ai/issues/161)) ([80707aa](https://github.com/farlleyferreira/conjecture-ai/commit/80707aa8089517b7a45d8dc26c6789eb31bdb91b))
* **main:** recover subcommand panics instead of crashing with a stack trace ([#168](https://github.com/farlleyferreira/conjecture-ai/issues/168)) ([397f256](https://github.com/farlleyferreira/conjecture-ai/commit/397f25620c5a06c65f610930817e9e77c2e05f18))
* move the unchanged clause from hypothesis to rule ([#31](https://github.com/farlleyferreira/conjecture-ai/issues/31)) ([4905de4](https://github.com/farlleyferreira/conjecture-ai/commit/4905de41a07fc9fef61810b75f6fc8ccc1dc0d2a))
* **omission-detection:** recognize otherwise as domain coverage in Type B ([#138](https://github.com/farlleyferreira/conjecture-ai/issues/138)) ([48639fb](https://github.com/farlleyferreira/conjecture-ai/commit/48639fb710bf9a1fbccfd1f96670d973d2434a8a))
* **omission-detection:** recognize Type C already covers state/event matrices ([#133](https://github.com/farlleyferreira/conjecture-ai/issues/133)) ([909eba3](https://github.com/farlleyferreira/conjecture-ai/commit/909eba38e80f2a1534a740238668916978d69d9a))
* **persistence:** move local data directory to a per-user path ([#151](https://github.com/farlleyferreira/conjecture-ai/issues/151)) ([bd99187](https://github.com/farlleyferreira/conjecture-ai/commit/bd991877d475bbae970b603bad57560ece021036))
* reject a second YAML document in yamlstrict.Load ([#21](https://github.com/farlleyferreira/conjecture-ai/issues/21)) ([edca6aa](https://github.com/farlleyferreira/conjecture-ai/commit/edca6aa526d5a897957efe1113e3fb11c4b3abd9))
* **verify:** check the recorded signature against the file's content ([#60](https://github.com/farlleyferreira/conjecture-ai/issues/60)) ([b5362dc](https://github.com/farlleyferreira/conjecture-ai/commit/b5362dc1b322d2f4fcee542afcab0cab658149c9))

## [1.0.1](https://github.com/farlleyferreira/conjecture-ai/compare/v1.0.0...v1.0.1) (2026-11-03)


### Bug Fixes

* drop per-commit commitlint check from CI ([#14](https://github.com/farlleyferreira/conjecture-ai/issues/14)) ([8bf8113](https://github.com/farlleyferreira/conjecture-ai/commit/8bf81133227da50ec722e9e71e48e383bd84b6c5))

# 1.0.0 (2026-11-03)


### Bug Fixes

* skip husky git hooks for semantic-release's automated commit ([#13](https://github.com/farlleyferreira/conjecture-ai/issues/13)) ([d5a6cf8](https://github.com/farlleyferreira/conjecture-ai/commit/d5a6cf81edec326bcd0bb9c13e1b1ca35452387e))


### Features

* add semantic commits, versioning, changelog and quality/security pipeline ([#1](https://github.com/farlleyferreira/conjecture-ai/issues/1)) ([4e7dd97](https://github.com/farlleyferreira/conjecture-ai/commit/4e7dd97ba47122acd9f43c3a39e8ec76d33d154e))

# Changelog

All notable changes to this project will be documented in this file.

The format is based on [Keep a Changelog](https://keepachangelog.com/en/1.1.0/),
and this project adheres to [Semantic Versioning](https://semver.org/spec/v2.0.0.html).

> This file is automatically updated by
> [release-please](https://github.com/googleapis/release-please) on every
> release, based on the project's [Conventional Commits](https://www.conventionalcommits.org/).
> Do not edit entries below by hand.

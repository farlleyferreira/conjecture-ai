🇧🇷 **Português** | [🇺🇸 English](README.en.md)

# conjecture-ai

**Escreva uma regra em palavras simples. O `conjecture` prova, de verdade,
se seu código obedece a ela.**

[![CI](https://github.com/farlleyferreira/conjecture-ai/actions/workflows/ci.yml/badge.svg)](https://github.com/farlleyferreira/conjecture-ai/actions/workflows/ci.yml)
[![License: MIT](https://img.shields.io/badge/license-MIT-blue.svg)](LICENSE)

## A forma mais simples de pensar nisso

Imagine uma regra de um jogo: *"se você tirar 6, joga de novo."* Você não
precisa ler o manual inteiro pra saber se isso é verdade - só precisa
jogar, tirar um 6, e ver o que acontece.

O `conjecture` faz isso com regras de software. Você escreve a regra uma
vez, em palavras simples, num arquivo pequeno - sem código. O `conjecture`
então joga o jogo por você: monta casos de teste reais a partir da sua
regra, executa seu programa *de verdade* contra eles dentro de uma sandbox
isolada, e te diz **conformant** (a regra se sustenta) ou **non-conformant**
(aqui está a entrada exata que quebrou). Ele nunca confia na palavra do
código. Ele sempre observa o que o código realmente faz.

## Diga uma vez, em palavras simples

Aqui está uma política real de companhia aérea: *"passagens Economy Flex
remarcadas com pelo menos 24 horas de antecedência não pagam taxa; o
resto paga."* Normalmente a única forma de saber se seu código realmente
faz isso é ler o código, ou escrever um teste unitário na mão e esperar
ter pensado nos casos certos.

```yaml
# conjecture/phenomena/ticketing/ticket-change-fee.cjtr
conjecture: 1.0.0

phenomenon: ticket-change-fee

concepts:
  hours before departure: {variable: input.hours_before_departure, type: decimal}
  fare class: {variable: input.fare_class, one_of: [economy-basic, economy-flex]}
  change fee: {variable: output.change_fee, type: decimal}
  free change window: {value: 24.00}

rules:
  flex fares waive the fee inside the window:
    id: flex-fee-waiver
    hypotheses:
      flex fare inside window is free:
        id: flex-inside-window
        given:
          - the fare class is economy-flex
          - the hours before departure is at least the free change window
        then: the change fee is 0.00
        otherwise: the change fee is at least 75.00
        examples:
          - {hours before departure: 30.00, fare class: economy-flex, change fee: 0.00}
          - {hours before departure: 10.00, fare class: economy-flex, change fee: 150.00}
          - {hours before departure: 30.00, fare class: economy-basic, change fee: 150.00}
```

Só isso. Sem código. Só a regra, em palavras que até alguém que não
programa conseguiria ler em voz alta.

## Veja ele pegar um bug real

Aqui está uma "implementação" com um bug real: ela esquece de checar a
classe da tarifa e só olha quantas horas faltam.

```python
# adapter.py
import json
data = json.load(open("conjecture_input.json"))
hours = float(data["input.hours_before_departure"])
fee = "0.00" if hours >= 24.00 else "150.00"
json.dump({"output.change_fee": fee}, open("conjecture_output.json", "w"))
```

```dockerfile
FROM python:3-slim
WORKDIR /work
COPY adapter.py /work/adapter.py
```

```
$ podman build -t ticket-fee-demo:latest .
$ conjecture check conjecture/phenomena/ticketing/ticket-change-fee.cjtr ticket-fee-demo:latest python3 /work/adapter.py
ticketing::ticket-change-fee::flex-fee-waiver: flex fare inside window is free: judge: accepted
ticketing::ticket-change-fee::flex-fee-waiver: flex fare inside window is free: conformance (directed sampling): non-conformant
ticketing::ticket-change-fee::flex-fee-waiver: flex fare inside window is free:   counterexample: map[input.fare_class:economy-basic input.hours_before_departure:30.00 output.change_fee:0.00]
```

Pego de verdade - uma passagem Economy Basic remarcada com 30 horas de
antecedência devia pagar a taxa, e essa implementação dispensa a taxa só
porque sobraram horas suficientes. Corrija o bug de verdade:

```python
# adapter.py
import json
data = json.load(open("conjecture_input.json"))
hours = float(data["input.hours_before_departure"])
fare_class = data["input.fare_class"]
fee = "0.00" if (fare_class == "economy-flex" and hours >= 24.00) else "150.00"
json.dump({"output.change_fee": fee}, open("conjecture_output.json", "w"))
```

```
$ podman build -t ticket-fee-demo:latest .
$ conjecture check conjecture/phenomena/ticketing/ticket-change-fee.cjtr ticket-fee-demo:latest python3 /work/adapter.py
ticketing::ticket-change-fee::flex-fee-waiver: flex fare inside window is free: judge: accepted
ticketing::ticket-change-fee::flex-fee-waiver: flex fare inside window is free: conformance (directed sampling): conformant

$ conjecture sign --user alice --confirmed conjecture/phenomena/ticketing/ticket-change-fee.cjtr
```

Escreva a regra uma vez, aponte pro código real, receba um veredito real -
e um registro assinado e atribuível de que um humano olhou pra isso.

## Por que o `conjecture`

- **Uma spec assinada congela antes do código ser julgado.** Depois que
  você assina um arquivo `.cjtr`, o juiz checa a implementação só contra
  essa regra exata e congelada - nada muda o gol de lugar no meio do
  jogo.
- **O modelo que escreve o código nunca corrige a própria prova.** Rascunho
  e implementação sempre rodam em sessões separadas. Nenhuma das duas
  consegue ver os casos ocultos que o juiz usa, chamar o juiz direto, ou
  assinar algo em seu nome.
- **Um modelo independente tenta ativamente quebrar sua regra.** Um SLM
  adversarial local por padrão, ou um swarm inteiro deles rodando em
  paralelo sem se coordenar ou debater entre si - caçando o contraexemplo
  que você não pensou.
- **Evidência se acumula - não é só pass/fail.** Uma regra sobe uma
  escada real de maturidade (Hypothesis → Thesis → Theory → Law), e o
  `conjecture` procura quatro tipos específicos de lacuna na spec antes
  que elas se tornem comportamento faltando no código em produção.
- **Todo contexto de verificação é versionado e retomável.** Liste,
  reabra uma versão passada exata, compare duas versões, ou restaure uma
  - nada de onde você parou precisa ser reconstruído de memória.
- **Funciona com a linguagem que você já tem.** O `conjecture` observa seu
  programa real só através de um contrato declarado de entrada/saída,
  nunca entendendo a semântica da sua linguagem.
- **Auto-ajuste fica sempre na coleira.** As únicas coisas que o
  `conjecture` pode otimizar sobre si mesmo - geração de entrada,
  orçamento do swarm, calibração - todas exigem seu `--approved` explícito
  antes de qualquer mudança.

## Começo rápido

> [!NOTE]
> Ainda não existe um download pronto - você mesmo compila o único
> binário. É um comando só, e é o mesmo comando em qualquer sistema.

Este repositório tem exatamente uma dependência CGO
(`github.com/smacker/go-tree-sitter`, usada pra análise de dependências),
então o binário precisa de um compilador C real pra qualquer sistema em
que for compilado:

```bash
CGO_ENABLED=1 go build -o conjecture ./cmd/conjecture
```

- **Linux/macOS**: funciona direto se `gcc`/`clang` estiver instalado.
- **Windows**: não existe gcc nativo. Compile dentro do WSL2 (instale
  `golang-go` e `gcc` lá), ou dentro de um container `golang` via
  Podman/Docker:
  ```bash
  podman run --rm -v "$(pwd):/src" -w /src golang:1.25 \
    go build -o conjecture ./cmd/conjecture
  ```

Você também precisa de **um runtime de container no `PATH` chamado
`docker`** - `conjecture check`/`implement`/`work` executam a implementação
testada dentro de um container com `--network none`. Docker real funciona
direto; Podman real funciona por trás de um shim de uma linha (`docker`
não vem apontado pra ele por padrão):

```bash
printf '#!/bin/sh\nexec podman "$@"\n' | sudo tee /usr/local/bin/docker
sudo chmod +x /usr/local/bin/docker
```

Depois:

```bash
conjecture init   # cria conjecture/phenomena/, conjecture/materializations/, .conjecture/
```

Escreva uma regra, rode `conjecture check` contra seu código real - esse é
todo o exemplo prático acima, do início ao fim.

> [!NOTE]
> `conjecture check`/`verify`/`sign`/`promote`/`context` não precisam de
> LLM nenhum - eles só executam regras já escritas contra código já
> escrito. Só `draft`/`implement`/`work`/`attack`/`swarm` precisam de um, e
> trocar de provedor é só uma flag `--provider` mais a API key daquele
> provedor - nunca muda código. Configure um destes: `ANTHROPIC_API_KEY`
> (provedor `anthropic`, o padrão), `DEEPSEEK_API_KEY` (provedor
> `deepseek`), um `ollama serve`/`llama-server` local (provedor
> `llamacpp`, sem key necessária), ou qualquer provedor que fale o formato
> Chat Completions da OpenAI - verificado contra nove: `openai`, `gemini`,
> `grok`, `llama-api` (API hospedada da Meta, diferente do `llamacpp`
> local), `mistral`, `cohere`, `bedrock`, `azure-openai`, `dashscope`
> (serve Qwen) - cada um só com suas próprias env vars, listadas em
> [`internal/llmclient/openaicompatible.go`](internal/llmclient/openaicompatible.go).

## Comandos

| Comando | Faz |
|---|---|
| `init` | Cria a estrutura do repositório (`conjecture/phenomena/`, `conjecture/materializations/`, `.conjecture/` no gitignore). |
| `draft` | Pede a um LLM pra rascunhar um spec `.cjtr` a partir de um pedido em linguagem natural. |
| `sign` | Registra autoria/confirmação num arquivo `.cjtr`; `--confirmed` também registra uma entrada real no ledger de decisões. |
| `verify` | Checa os exemplos próprios de um spec contra suas próprias hipóteses - sem sandbox, sem implementação. |
| `check` | Executa o ciclo de conformidade por amostragem direcionada de uma hipótese contra uma implementação real em sandbox. |
| `implement` / `work` | Pede a um LLM pra escrever/corrigir código de implementação até ficar conforme (ou o orçamento de rodadas acabar); `work` consome da fila em vez de receber um arquivo direto. |
| `attack` / `swarm` | Executa um ou vários ataques adversariais de LLM contra uma hipótese, procurando um contraexemplo. |
| `promote` | Promove uma hipótese/law pra Theory ou Law quando suas condições reais de evidência se cumprem; registra uma entrada no ledger de decisões. |
| `law` | Resolve um conflito entre uma law definida pelo usuário e uma prova formal ou outra law. |
| `context` | `list`/`resume`/`open`/`origin`/`diff`/`restore`/`resolve-conflict` sobre um contexto de verificação salvo. |
| `base` | Conecta/busca uma base de conhecimento (normas, definições, referências) com escopo de projeto ou área. |
| `queue` | A fila de trabalho de três níveis (blocker/current-task/background) que `work` consome; `review` lista itens que não puderam ser tratados automaticamente. |
| `results` | Importa resultados já produzidos em JUnit XML ou TAP de um sistema alvo. |
| `graph` | Lê/define peso/relevância/confiança de um nó do motor de contexto. |
| `optimize` | Aplica uma proposta de calibração - sempre exige `--approved` explícito. |
| `settings` | Checa um valor contra um piso configurado pelo time (nunca um teto). |
| `translate` | Gera o comentário determinístico `# pt:` de uma hipótese. |
| `deps` | Checa código Go por imports proibidos. |
| `watch` | Monitora um diretório e reverifica qualquer arquivo `.cjtr` que mudar. |
| `mcp` | Executa o conjecture como um servidor MCP. |

Rode qualquer comando sem argumentos (ou qualquer bobagem no formato
`--help`) pra ver sua própria string de uso real.

<details>
<summary><strong>Escopo e limites</strong> - leia antes de confiar nisso pra algo que importa</summary>

O núcleo de verificação (check → attack/swarm → promote, execução em
sandbox, o ledger de decisões, o conjunto completo de comandos de
composição de contexto) está implementado e já foi rodado de verdade:
contra um solver de restrições real (z3), um LLM local real
(llama.cpp/Ollama), uma biblioteca de terceiros genuína e real
(python-slugify - isso achou e corrigiu um bug real neste repositório), e
verificação real de isolamento de rede.

Pra que isso não serve, de propósito:

- **Não é um framework de testes genérico.** Ele só checa se uma
  implementação em execução corresponde a uma regra que você escreveu na
  gramática restrita do `.cjtr` (comparações sobre decimais/categorias,
  `given`/`then`/`otherwise`/`always`). Não consegue expressar lógica
  arbitrária, e não vai substituir seus testes unitários pra nada que não
  tenha o formato de uma regra de negócio declarada.
- **Não é prova exaustiva, por padrão.** O modo de amostragem direcionada
  do `check` tenta valores de fronteira e partição, não cada entrada
  possível. Só uma hipótese sobre um domínio totalmente delimitado e
  enumerável recebe o nível de rigor de "prova completa" deste código - a
  maioria dos checks é evidência forte, não uma garantia matemática pra
  toda entrada possível.
- **Não é um produto de sandboxing/segurança.** A sandbox bloqueia acesso
  de rede de saída e nada mais - sem hardening de filesystem ou de
  recursos. Não aponte pra código em que você já não confia pra rodar na
  sua máquina.
- **Não é um analisador estático pra um codebase inteiro.** `deps check`
  é um checker estreito de imports proibidos pra um arquivo Go por vez,
  não um linter.
- **Ainda não tem histórico de produção.** Este é um projeto experimental,
  construído majoritariamente numa única sessão longa. Ninguém ainda usou
  pra tomar uma decisão real sobre código real do qual dependa - esse é o
  próximo teste real.

Mais duas coisas específicas pra saber:

- **O driver de acessibilidade de GUI tem backends reais pra Windows (UI
  Automation) e Linux (AT-SPI), mas a gramática `.cjtr` ainda não tem como
  declarar um alvo de GUI** - a capacidade existe e é testada isoladamente,
  mas nada no fluxo principal de verificação da CLI consegue alcançá-la.
  macOS não é suportado (sem backend, por decisão deliberada - veja
  [issue #144](https://github.com/farlleyferreira/conjecture-ai/issues/144)).
- **Geração de conteúdo de materialização** (um caminho
  `conjecture/materializations/` existe depois do `conjecture init`, mas
  nada escreve nada lá ainda) está explicitamente fora do escopo do
  trabalho de persistência feito até agora.

</details>

## Licença

Este projeto é licenciado sob a [Licença MIT](LICENSE).

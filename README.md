🇧🇷 **Português** | [🇺🇸 English](README.en.md)

# conjecture-ai

**Seu agente disse "pronto". O `conjecture` tenta derrubar o que ele
escreveu, usando uma regra que você aprovou, e mostra a entrada exata
que quebrou.**

[![CI](https://github.com/farlleyferreira/conjecture-ai/actions/workflows/ci.yml/badge.svg)](https://github.com/farlleyferreira/conjecture-ai/actions/workflows/ci.yml)
[![License: MIT](https://img.shields.io/badge/license-MIT-blue.svg)](LICENSE)

> **Projeto experimental.** Ninguém usou isto ainda para decidir algo
> sobre código do qual dependa. Leia [Limites](#limites) antes de confiar.

## Em 30 segundos

A regra: *"Economy Flex remarcada com 24h ou mais de antecedência não
paga taxa. O resto paga."*

O agente entregou isto e disse que terminou:

```python
hours = float(data["input.hours_before_departure"])
fee = "0.00" if hours >= 24.00 else "150.00"
```

O `conjecture` monta casos a partir da regra, roda o programa num
container sem rede e responde:

```
$ conjecture check conjecture/phenomena/ticketing/ticket-change-fee.cjtr ticket-fee-demo:latest python3 /work/adapter.py
ticketing::ticket-change-fee::flex-fee-waiver: flex fare inside window is free: judge: accepted
ticketing::ticket-change-fee::flex-fee-waiver: flex fare inside window is free: conformance (directed sampling): non-conformant
ticketing::ticket-change-fee::flex-fee-waiver: flex fare inside window is free:   counterexample: map[input.fare_class:economy-basic input.hours_before_departure:30.00 output.change_fee:0.00]
```

Uma Economy Basic com 30 horas de antecedência saiu de graça: o código
esqueceu a classe da tarifa. Depois da correção:

```
$ conjecture check conjecture/phenomena/ticketing/ticket-change-fee.cjtr ticket-fee-demo:latest python3 /work/adapter.py
ticketing::ticket-change-fee::flex-fee-waiver: flex fare inside window is free: judge: accepted
ticketing::ticket-change-fee::flex-fee-waiver: flex fare inside window is free: conformance (directed sampling): conformant

$ conjecture status conjecture/phenomena/ticketing/ticket-change-fee.cjtr
ticketing::ticket-change-fee: not signed
ticketing::ticket-change-fee::flex-fee-waiver::flex-inside-window: stage=Hypothesis attacked="directed sampling" survived="every generated case conformed"
```

O `status` não diz "aprovado". Diz o que foi atacado, o que resistiu, em
que estágio a regra está e que nenhum humano assinou ainda.

<details>
<summary>A regra como arquivo <code>.cjtr</code></summary>

É YAML numa gramática restrita. Seu agente escreve, você revisa e assina.

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

</details>

<details>
<summary>O programa completo e o container da demonstração</summary>

O `conjecture` não lê seu código. Ele grava `conjecture_input.json`,
executa o comando que você indicar dentro da imagem e lê
`conjecture_output.json`.

```python
# adapter.py - versão corrigida
import json
data = json.load(open("conjecture_input.json"))
hours = float(data["input.hours_before_departure"])
fare_class = data["input.fare_class"]
fee = "0.00" if (fare_class == "economy-flex" and hours >= 24.00) else "150.00"
json.dump({"output.change_fee": fee}, open("conjecture_output.json", "w"))
```

```dockerfile
FROM python:3-slim
WORKDIR /work
COPY adapter.py /work/adapter.py
```

```bash
podman build -t ticket-fee-demo:latest .
```

</details>

## O problema

O mesmo modelo que escreve o código escreve os testes e declara que
terminou. Um segundo agente do mesmo modelo revisando herda os mesmos
pontos cegos. E "passou" não diz o que foi testado nem o que ficou de
fora.

O `conjecture` separa os papéis: a regra é assinada por você, os casos
são gerados a partir dela por um verificador sem IA, e o veredito vem de
executar o programa.

## Instalar

Linux e macOS:

```bash
curl -fsSL https://raw.githubusercontent.com/farlleyferreira/conjecture-ai/main/install.sh | sh
```

Windows (PowerShell):

```powershell
irm https://raw.githubusercontent.com/farlleyferreira/conjecture-ai/main/install.ps1 | iex
```

Os scripts baixam o binário da [última release](https://github.com/farlleyferreira/conjecture-ai/releases/latest)
e conferem o SHA-256. Com Go instalado, também funciona:

```bash
go install github.com/farlleyferreira/conjecture-ai/cmd/conjecture@latest
```

Você precisa de Docker ou Podman para `check`. O `conjecture` usa
`docker` se existir no `PATH`, senão `podman`. Para escolher outro,
defina `CONJECTURE_CONTAINER_RUNTIME`.

Os binários não são assinados. O Windows (SmartScreen ou Smart App
Control) pode avisar ou bloquear na primeira execução.

## Usar com seu agente

O `conjecture` roda como servidor MCP e instala uma skill que ensina o
fluxo ao agente. Com Claude Code:

```bash
conjecture init
```

```bash
claude mcp add conjecture -- conjecture mcp
```

O `init` cria `conjecture/phenomena/` e
`.claude/skills/conjecture-workflow/SKILL.md`. Depois, peça a regra em
linguagem natural. O fluxo que a skill impõe:

1. O agente escreve o `.cjtr` com você e chama `verify` a cada mudança.
2. Você assina no seu terminal: `conjecture sign --user <nome> --confirmed <arquivo.cjtr>`.
   O servidor MCP não expõe `sign` nem `promote`: o agente não consegue
   aprovar a regra por você.
3. O agente escreve a implementação e chama `check`.
4. Se vier contraexemplo, ele corrige o código. A regra assinada não muda
   para fazer o código passar.
5. `status` resume quem assinou e o estado de cada hipótese.

Qualquer cliente MCP por stdio funciona com `conjecture mcp`. A skill
segue o formato do Claude Code.

## Usar pelo terminal

| Comando | Faz |
|---|---|
| `conjecture verify <arquivo.cjtr>` | Confere os exemplos da regra contra a própria regra. Sem container, sem IA. |
| `conjecture check <arquivo.cjtr> <imagem> <comando...>` | Gera casos, executa o programa no container e dá o veredito. Sem IA. |
| `conjecture sign --user <nome> --confirmed <arquivo.cjtr>` | Registra que um humano revisou a regra. |
| `conjecture status [arquivo.cjtr]` | Mostra assinatura, estágio e evidência de cada hipótese. |

`verify`, `check`, `sign`, `status`, `promote` e `context` rodam na sua
máquina e não chamam modelo nenhum. `draft`, `implement`, `work`,
`attack` e `swarm` chamam o provedor que você configurar. Com um
provedor de nuvem, o que você enviar a ele sai da sua máquina; com
`ollama` ou `llama-server` local, não sai.

Lista completa de comandos e provedores: [docs/COMMANDS.pt-BR.md](docs/COMMANDS.pt-BR.md).

## Como a confiança sobe

Cada regra tem um estágio: Hypothesis, Thesis, Theory, Law. A ideia é
que subir exija evidência nova, não repetir o mesmo teste. O que existe
hoje é menos que isso:

| Passagem | Como acontece hoje |
|---|---|
| Hypothesis para Thesis | Automática no `check`, mas só para regras declaradas como `laws`, depois de 3.000 casos acumulados sem falha. Um contraexemplo zera a contagem. Uma hipótese comum continua em Hypothesis mesmo com `check` conforme. |
| Thesis para Theory | Manual, com `conjecture promote theory`. As condições (mecanismo confirmado, reprodução por outra família de modelo) são flags que você atesta. |
| Theory para Law | Manual, com `conjecture promote law`. As condições (amostras ocultas, versões novas do código) também são flags que você atesta. |

Toda promoção fica registrada com quem decidiu.

## Limites

- **Não prova que o código está certo.** O `check` testa valores de
  fronteira e de partição, não toda entrada possível. Só um domínio
  totalmente enumerável recebe verificação completa.
- **Só cobre regras de negócio declaráveis.** A gramática do `.cjtr`
  expressa comparações sobre decimais e categorias
  (`given`/`then`/`otherwise`/`always`). Não substitui teste unitário de
  lógica arbitrária.
- **Você precisa empacotar o programa.** Cada implementação vira uma
  imagem de container com um adapter que lê e grava JSON. É trabalho a
  mais por regra.
- **O adapter é um ponto de trapaça.** Se o mesmo agente escreve o
  código e o adapter, ele pode fazer o adapter responder o que a regra
  espera. Revise o adapter.
- **A regra vale o quanto a sua revisão vale.** Um `.cjtr` rascunhado por
  IA e não assinado gera aviso no `verify` e aparece como `not signed`
  no `status`, mas o `conjecture` não sabe se a regra assinada é a regra que você queria.
- **A escada de estágios está incompleta.** Hipóteses comuns não sobem
  de estágio, e Theory e Law dependem de flags que um humano atesta, não
  de medição. Veja [Como a confiança sobe](#como-a-confiança-sobe).
- **`implement` e `work` não alteram seu projeto.** Eles geram um script
  Python descartável de um arquivo só. Para código de verdade, use o
  fluxo com o seu agente.
- **`attack` e `swarm` não executam código.** O contraexemplo que um
  modelo adversário alega não é verificado mecanicamente. Leia antes de
  acreditar.
- **A sandbox só bloqueia rede de saída.** Não há isolamento de
  arquivos nem de recursos. Não rode código em que você não confia.
- **Binários de Linux e Windows não têm `deps check`.** Esse comando usa
  Tree-sitter, que é código C. Para usá-lo, compile com `CGO_ENABLED=1`.
- **Sem histórico de produção.** O projeto foi construído em grande
  parte numa única sessão longa, com agentes de IA.
- **Alvos de GUI não funcionam pelo fluxo principal.** Existem backends
  de acessibilidade para Windows e Linux, mas a gramática ainda não
  declara alvo de GUI. macOS não é suportado
  ([issue #144](https://github.com/farlleyferreira/conjecture-ai/issues/144)).

## Como saberemos se funciona

Ainda não sabemos. A meta decisiva foi fixada antes do experimento: pelo
menos 30% menos violações do que a mesma IA sem o `conjecture`, em 50
tarefas de cada um de dois projetos em linguagens diferentes. Se essa
meta falhar, nenhuma outra salva a proposta.

Resultado: ainda não publicado.

## Compilar do código

```bash
go build -o conjecture ./cmd/conjecture
```

Não precisa de compilador C. Com `CGO_ENABLED=1` e `gcc` ou `clang`, o
binário também inclui `deps check`.

## Contribuir

Veja [CONTRIBUTING.md](CONTRIBUTING.md).

## Licença

[Licença MIT](LICENSE).

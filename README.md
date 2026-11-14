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
def change_fee(inputs):
    hours = Decimal(inputs["hours_before_departure"])
    fee = "0.00" if hours >= 24 else "150.00"
    return {"change_fee": fee}
```

```
$ conjecture check conjecture/phenomena/ticketing/ticket-change-fee.cjtr --lang python --entry pricing.fees:change_fee
ticketing::ticket-change-fee::flex-fee-waiver: flex fare inside window is free: judge: accepted
ticketing::ticket-change-fee::flex-fee-waiver: flex fare inside window is free: conformance (directed sampling): non-conformant
ticketing::ticket-change-fee::flex-fee-waiver: flex fare inside window is free:   counterexample: fare_class=economy-basic, hours_before_departure=30.00 -> change_fee=0.00
```

Uma Economy Basic saiu de graça: o código esqueceu a classe da tarifa.
O agente corrigiu, mas escreveu `hours > 24`:

```
ticketing::ticket-change-fee::flex-fee-waiver: flex fare inside window is free: conformance (directed sampling): non-conformant
ticketing::ticket-change-fee::flex-fee-waiver: flex fare inside window is free:   counterexample: fare_class=economy-flex, hours_before_departure=24.00 -> change_fee=150.00
```

Exatamente 24 horas devia ser grátis. Depois da segunda correção:

```
ticketing::ticket-change-fee::flex-fee-waiver: flex fare inside window is free: conformance (directed sampling): conformant
ticketing::ticket-change-fee::flex-fee-waiver: flex fare inside window is free: conformance (hidden sampling (20 random cases, seed 17164254410232975056)): conformant
ticketing::ticket-change-fee::flex-fee-waiver: flex fare inside window is free: 20/100 failure-free hidden cases against this code (low weight)

$ conjecture status conjecture/phenomena/ticketing/ticket-change-fee.cjtr
ticketing::ticket-change-fee: not signed
ticketing::ticket-change-fee::flex-fee-waiver::flex-inside-window: stage=Hypothesis attacked="hidden sampling (20 random cases, seed 17164254410232975056)" survived="every sampled case conformed"
ticketing::ticket-change-fee::flex-fee-waiver::flex-inside-window: hidden cases 20/100 (low weight); code versions meeting it: 0; model families meeting it: none yet
```

O `status` não diz "aprovado". Diz o que foi atacado, o que resistiu,
quantos casos faltam para subir de estágio e que nenhum humano assinou
ainda.

Nessa demonstração não há Dockerfile, imagem nem adapter escritos à mão.
As saídas acima são de uma execução de verdade.

<details>
<summary>A regra como arquivo <code>.cjtr</code></summary>

É YAML numa gramática restrita. Seu agente escreve, você revisa e assina.

```yaml
# conjecture/phenomena/ticketing/ticket-change-fee.cjtr
conjecture: 1.0.0

phenomenon: ticket-change-fee

concepts:
  hours before departure: {variable: input.hours_before_departure, type: decimal, range: {min: 0.00, max: 720.00}}
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

Cada binário tem atestação de proveniência. Para conferir um download:

```bash
gh attestation verify conjecture_linux_amd64.tar.gz --repo farlleyferreira/conjecture-ai
```

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
3. O agente escreve a implementação e uma função de entrada para a regra.
4. O agente chama `check`. Se vier contraexemplo, ele corrige o código.
   A regra assinada não muda para fazer o código passar.
5. `status` resume quem assinou e o estado de cada hipótese.

Qualquer cliente MCP por stdio funciona com `conjecture mcp`. A skill
segue o formato do Claude Code.

## A função de entrada

O `conjecture` não lê seu código. Você aponta uma função, e ele gera o
adapter, escolhe a imagem da linguagem e monta seu projeto somente para
leitura, sem rede.

A função tem o mesmo formato em toda linguagem: recebe as entradas da
regra como mapa de nome para texto e devolve as saídas do mesmo jeito.
Os valores chegam como texto (`"30.00"`), para um decimal não virar
ponto flutuante no caminho.

| Linguagem | `--lang` | `--entry` |
|---|---|---|
| Python | `python` | `pacote.modulo:funcao` |
| JavaScript | `javascript` | `caminho/arquivo.js:funcaoExportada` |
| TypeScript | `typescript` | `caminho/arquivo.ts:funcaoExportada` |
| Go | `go` | `./diretorio/do/pacote:Funcao` |
| Java | `java` | `com.exemplo.Classe:metodoEstatico` |
| C# | `csharp` | `caminho/Assembly.dll:Namespace.Classe:MetodoEstatico` |
| Ruby | `ruby` | `caminho/arquivo.rb:Modulo::Classe.metodo` |

Para outra linguagem, ou para dependências que não estão na árvore do
projeto, existe a forma original: `conjecture check <arquivo.cjtr> <imagem> <comando...>`,
com um programa que lê `conjecture_input.json` e grava
`conjecture_output.json`.

## Usar pelo terminal

| Comando | Faz |
|---|---|
| `conjecture verify <arquivo.cjtr>` | Confere os exemplos da regra contra a própria regra. Sem container, sem IA. |
| `conjecture check <arquivo.cjtr> --lang <linguagem> --entry <entrada>` | Gera casos, executa sua função no container e dá o veredito. Sem IA. |
| `conjecture sign --user <nome> --confirmed <arquivo.cjtr>` | Registra que um humano revisou a regra. |
| `conjecture status [arquivo.cjtr]` | Mostra assinatura, estágio, evidência e o que falta para a próxima promoção. |

`verify`, `check`, `sign`, `status`, `promote` e `context` rodam na sua
máquina e não chamam modelo nenhum. `draft`, `implement`, `work`,
`attack` e `swarm` chamam o provedor que você configurar. Com um
provedor de nuvem, o que você enviar a ele sai da sua máquina; com
`ollama` ou `llama-server` local, não sai.

Lista completa de comandos e provedores: [docs/COMMANDS.pt-BR.md](docs/COMMANDS.pt-BR.md).

## Como a confiança sobe

Cada regra tem um estágio. Nenhuma passagem aceita a palavra de quem
pede: todas são calculadas do que está registrado.

O `check` roda dois tipos de caso. Os direcionados testam as bordas do
domínio e os limites que a própria regra cita (24.00, 23.99, 24.01). Os
ocultos são sorteados do domínio declarado a cada execução. Só os
ocultos contam para promoção: repetir os mesmos casos cem vezes é uma
observação, não cem.

| Passagem | O que precisa estar registrado |
|---|---|
| Hypothesis para Thesis | Casos ocultos sem falha contra a regra e o código atuais: 100 (peso baixo), 300, 1.000 ou 3.000 (peso crítico). Automática no `check`. |
| Thesis para Theory | Todas as regras do arquivo em Thesis, a regra assinada com `--confirmed`, a contagem atingida em implementações de duas famílias de modelo diferentes, e um ataque adversário sem violação. Via `conjecture promote theory`. |
| Theory para Law | Regra universal (`always`), em Theory, com a contagem atingida em duas versões do código. Via `conjecture promote law`. |

Um contraexemplo rebaixa a regra e zera a contagem daquela versão do
código. Mudar um byte do código começa uma contagem nova.

## Limites

- **Não prova que o código está certo.** Os casos são amostras. 100
  casos sem falha limitam a taxa de violação a cerca de 3% com 95% de
  confiança, dentro do domínio declarado e não fora dele. Só um domínio
  totalmente enumerável recebe verificação completa.
- **Só cobre regras de negócio declaráveis.** A gramática do `.cjtr`
  expressa comparações sobre decimais e categorias
  (`given`/`then`/`otherwise`/`always`). Não substitui teste unitário de
  lógica arbitrária.
- **A função de entrada é um ponto de trapaça.** O adapter é gerado, mas
  a função de entrada é código do projeto e decide o que os nomes da
  regra significam. Se ela reimplementar a regra em vez de chamar o
  código real, o `conjecture` aprova a função, não o sistema. Revise.
- **A regra vale o quanto a sua revisão vale.** Um `.cjtr` rascunhado por
  IA e não assinado gera aviso no `verify` e aparece como `not signed`
  no `status`, mas o `conjecture` não sabe se a regra assinada é a regra
  que você queria.
- **A família do modelo é declarada, não verificada.** `--produced-by`
  registra quem escreveu a implementação. Nada no código prova o autor.
  A passagem para Theory depende dessa declaração.
- **Sem domínio declarado, não há casos ocultos.** Um decimal sem
  `range` ou um texto sem `one_of` não pode ser sorteado, e a regra fica
  em Hypothesis. O `check` avisa qual conceito falta.
- **Dependências precisam estar na árvore do projeto.** O container roda
  sem rede. `node_modules`, classes compiladas e o cache de módulos do Go
  funcionam; um `pip install` global não. Para o resto, passe `--image`.
- **TypeScript roda pelo Node, sem compilador.** Imports relativos
  precisam de extensão explícita. C# exige que você compile o projeto
  antes e aponte a DLL.
- **Cada caso é uma execução de container.** Medimos cerca de meio
  segundo por caso (Python, Podman no Windows): 100 casos em torno de um
  minuto, 3.000 em torno de 25. Linguagens compiladas e outras máquinas
  não foram medidas.
- **`implement` e `work` não alteram seu projeto.** Eles geram um script
  Python descartável de um arquivo só. Para código de verdade, use o
  fluxo com o seu agente.
- **`attack` e `swarm` não executam código.** O contraexemplo que um
  modelo adversário alega não é verificado mecanicamente. Leia antes de
  acreditar.
- **A sandbox só bloqueia rede de saída.** Não há isolamento de
  arquivos nem de recursos. Não rode código em que você não confia.
- **Os binários não têm assinatura de código.** A atestação prova a
  origem, mas o Windows (SmartScreen, Smart App Control, Defender) pode
  avisar, bloquear ou marcar o executável como suspeito.
- **Binários de Linux e Windows não têm `deps check`.** Esse comando usa
  Tree-sitter, que é código C. Para usá-lo, compile com `CGO_ENABLED=1`.
- **Sem histórico de produção.** O projeto foi construído em grande
  parte em sessões longas com agentes de IA.
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

# Comandos

[English](COMMANDS.md)

Rode qualquer comando sem argumentos para ver a linha de uso dele.

| Comando | Faz |
|---|---|
| `init` | Cria a estrutura do repositório (`conjecture/phenomena/`, `conjecture/materializations/`, `.conjecture/` no gitignore). |
| `draft` | Pede a um LLM pra rascunhar um spec `.cjtr` a partir de um pedido em linguagem natural. |
| `sign` | Registra autoria/confirmação num arquivo `.cjtr`; `--confirmed` também registra uma entrada real no ledger de decisões. |
| `verify` | Checa os exemplos próprios de um spec contra suas próprias hipóteses - sem sandbox, sem implementação. |
| `check` | Executa o ciclo de conformidade por amostragem direcionada de uma hipótese contra uma implementação real em sandbox. |
| `status` | Mostra assinatura, estágio e evidência registrada de um arquivo `.cjtr`, ou os totais do projeto sem argumento. |
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

`deps` só existe em binários compilados com `CGO_ENABLED=1` (os de
macOS das releases, ou os que você compilar com um compilador C). Nos
demais, ele informa que precisa de um build com CGO.

## Quais comandos chamam um modelo

`verify`, `check`, `sign`, `status`, `promote` e `context` não chamam
modelo nenhum. `draft`, `implement`, `work`, `attack` e `swarm` chamam.

## Provedores

Escolha com `--provider`. O padrão é `anthropic`.

| Provedor | Configuração |
|---|---|
| `anthropic` | `ANTHROPIC_API_KEY` |
| `deepseek` | `DEEPSEEK_API_KEY` |
| `llamacpp` | Um `ollama serve` ou `llama-server` local. Sem chave. |
| `openai`, `gemini`, `grok`, `llama-api`, `mistral`, `cohere`, `bedrock`, `azure-openai`, `dashscope` | Formato Chat Completions da OpenAI. Variáveis de ambiente de cada um em [`internal/llmclient/openaicompatible.go`](../internal/llmclient/openaicompatible.go). |

## Runtime de container

`check`, `implement` e `work` executam a implementação num container com
`--network none`. O `conjecture` usa `docker` se estiver no `PATH`,
senão `podman`. `CONJECTURE_CONTAINER_RUNTIME` escolhe outro executável.

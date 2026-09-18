# Métricas del núcleo de ditto — antes y después

Artefacto pedido al abrir este trabajo: *sacar las métricas actuales, guardarlas, y después las mejoras*. Cada número de acá está medido, y cada uno lleva su condición: sin la condición, un ratio no dice nada.

- Rama: `perf/module-scope-core`, **sin push**.
- Punto de partida: `5f65e3d`.
- Registro completo con el porqué: [`performance-core-log.md`](./performance-core-log.md).
- Notas de experimento: [`experiments/`](./experiments/).

## Reglas de lectura

Exactitud e intervalo son cosas distintas:

- **Contadores** son enteros, idénticos en cualquier máquina, y son el contrato. El repositorio los usa como puerta.
- **Tiempo de pared** se reporta y nunca decide. La máquina nunca está ociosa.
- Un **ratio** se toma dentro de una ventana, con el orden de modos rotado y un calentamiento descartado. Dos absolutos medidos con minutos de diferencia no son una medición.
- Un ratio **sin la forma del repositorio al lado** no significa nada: la misma ruta da 0,32 o 0,56 según la forma del módulo.

## 1. Contadores del repositorio, guardados

Contra el fixture sintético de seis archivos, salvo el último, que se mide contra el árbol de ditto.

| Contador | Antes | Después |
|---|---:|---:|
| `sourceParsesPerReleaseWithThreeViruses` | 4 | 4 |
| `astWalksPerReleaseWithThreeViruses` | 12 | 12 |
| `laboratoryRunsPerReleaseWholeFixture` | 48 | 48 |
| `testCommandInvocationsPerReleaseWholeFixture` | 49 | 49 |
| `filesLinkedPerSandbox` | 6 | 6 |
| `sandboxesBuiltPerRelease` | 1 | 1 |
| `laboratoryRunsForOneChangedFunction` | 4 | 4 |
| `laboratoryRunsForOneChangedFunctionInEachOfTwoFiles` | 8 | 8 |
| `mutantsPerReleaseOnThisRepository` | 789 | **850** |

Los ocho primeros **no se movieron**, y eso es el resultado: la ruta optimizada es opt-in y no toca el camino que esos contadores miden.

El noveno subió, y cada salto está atribuido por archivo, no al cambio entero:

| Salto | Causa | Archivos |
|---|---:|---|
| 789 → 813 (+24) | núcleo de alcance modular | `module_scope.go` +23, `gatedlaboratory.go` +1, `options.go` +0 |
| 813 → 818 (+5) | el motivo del veredicto en la ruta modular | `module_scope.go` 23 → 28 |
| 818 → 846 (+28) | la clausura de observabilidad | `module_scope.go` 28 → 54, `gatedlaboratory.go` 37 → 39 |
| 846 → 850 (+4) | un sandbox y un directorio de compilación por release | `gatedlaboratory.go` 39 → 42, `module_scope.go` 54 → 55 |

Cada suma coincide con lo que reportó el ratchet.

## 2. El mecanismo, aislado

Fixture de 3 paquetes y 12 mutantes. Contadores exactos, sin tiempo de pared.

| Modo | Arranques del driver | Ejecuciones de paquetes | Veredictos |
|---|---:|---:|---|
| A — ordinario | 13 | 52 | 6 killed / 6 survived |
| B — sólo paquete (lo que había) | 1 | **13** | 5 killed / **7 survived** |
| C — alcance modular | **1** | **52** | 6 killed / 6 survived |

Tres rondas rotadas del ratio C/A: **0,1951 · 0,1993 · 0,1809**.

La fila B es el defecto: hacía 13 ejecuciones donde el usuario pidió 52. Un mutante que sólo mata un paquete dependiente sobrevivía ahí y moría en A y C. **Su velocidad venía de omitir pruebas configuradas.**

## 3. De punta a punta, con el binario real

Módulo de 3 paquetes, 30 mutantes. Un calentamiento descartado, tres rondas con el orden rotado, razón calculada dentro de cada ronda.

| Ronda | Orden | Ordinario | `--gated` | Razón |
|---|---|---:|---:|---:|
| 1 | A B | 22.959 ms | 7.349 ms | **0,3201** |
| 2 | B A | 23.074 ms | 7.266 ms | **0,3149** |
| 3 | A B | 23.190 ms | 7.424 ms | **0,3201** |

Dispersión 1,7%. Veredictos idénticos y las direcciones de supervivientes **byte a byte iguales** sobre seis reportes reales.

## 4. Módulo de 10 paquetes, 40 mutantes

| Momento | Ordinario | `--gated` | Razón |
|---|---:|---:|---:|
| Antes de la clausura | 64.103 ms | 42.686 ms | **0,6649** |
| Después de la clausura | 63.619 / 63.560 / 63.414 ms | 25.546 / 25.454 / 25.618 ms | **0,4015 / 0,4005 / 0,4040** |
| Después de un sandbox y un directorio de compilación por release | 64.583 / 65.964 / 64.006 ms | 14.620 / 14.708 / 14.798 ms | **0,2264 / 0,2230 / 0,2312** |

La clausura bajó el ratio de 0,6649 a ~0,40 y la ganancia pasó de 1,50× a **~2,50×**.

**El techo de la clausura, medido aparte:** en un fixture de 7 paquetes, 3 de 7 binarios observan al paquete mutado; 35 → 15 ejecuciones. Las 4 islas no corren, y `top` entra aunque nunca nombra a `base`.

## 5. La forma del repositorio decide el ratio

Cadena de 8 paquetes, 6 mutantes. Misma generación, sólo cambia dónde están los sitios mutables.

| Fixture | Observadores | Ronda 1 | Ronda 2 | Ronda 3 |
|---|---:|---:|---:|---:|
| Mutación en la cabeza | 8 de 8 | 0,3492 | 0,3528 | 0,3597 |
| Mutación en la cola | 1 de 8 | 0,5408 | 0,5623 | 0,5536 |

**La cabeza, donde la clausura no quita nada, es 1,5× más rápida.** La predicción decía lo contrario y fue refutada.

La causa, medida después: `ditto.Release` agrupa por archivo y **cada tanda paga una compilación module-wide**. La cabeza tenía mutantes en un archivo (1 compilación, 1.651 ms); la cola en dos (1.676 + 1.580 ms). Esa segunda compilación es toda la brecha de 1,9 s.

## 6. El motivo del veredicto

Un binario de pruebas no puede emitir `go test -json`: esa bandera es del driver, no del binario.

| Modo | Motivo reportado |
|---|---|
| Control: `go test -count=1 -json ./...` | `assertion` |
| Ruta modular, antes | **`unknown`** |
| Ruta modular vía `go tool test2json` | `assertion` |
| Paquete que no compila | sin motivo: falla cerrado y el diagnóstico es la salida |

Con `unknown`, `internal/confirminglaboratory` — que sólo reejecuta un kill cuando el motivo es `assertion` — nunca reejecutaba nada: **`--confirm-kills` era un no-op silencioso en la ruta optimizada**.

## 7. El costo de la clausura

Un `go list -deps -test -json ./...` por release: **190, 177, 174 ms**. Se paga una vez, no por mutante.

## 8. Lo que se midió y no se cobró

Compartir un directorio de compilación entre las tandas de un release:

| Escenario | Total |
|---|---:|
| 10 directorios frescos (lo de hoy) | 15.466 ms |
| 1 directorio compartido | 3.087 ms |
| **Premio aparente** | **12.379 ms** |

Contra umbral pre-registrado de 8.500 ms: el premio existía. **Se implementó, se midió, y no apareció.**

| Momento | Razón en 10 paquetes |
|---|---|
| Antes | 0,4005 · 0,4040 · 0,3995 |
| Con el directorio compartido | 0,4031 · 0,3996 · 0,4015 |

**Causa:** el directorio compartido sí se usa, pero cada tanda enlaza **su propio sandbox**, las rutas absolutas difieren, y los build IDs de Go las incluyen. Nada queda al día entre tandas, así que la comprobación de vigencia que el bucle de precio estaba midiendo nunca llega a dispararse. Los 12.379 ms eran una medición real **de una situación que no puede ocurrir**.

**Ese cambio se revirtió** — agregaba tres mutantes y un contador a cambio de nada medible. Pero el precio era real, y la causa de que no apareciera también: cada tanda enlazaba su propio sandbox y los build IDs de Go incluyen esas rutas. Con **un sandbox y un directorio por release**, las dos mitades juntas, el premio llegó: el gated bajó de 25,6 s a 14,6 s sobre el mismo fixture, y el ratio de 0,40 a **0,2230–0,2312** (commit `7e30910`).

## 9. Lo que NO está medido

Decirlo sin adornos es parte del artefacto:

| Hueco | Estado |
|---|---|
| Ratio con suite pesada | Estimación previa 0,50–0,58, no remedido en esta rama |
| Kill por **deadline** en la ruta modular | El reloj es de `-test.timeout`; su pánico se convierte en `Assertion`. Pregunta separada, sin arreglar |
| El gate propio de este repositorio | Tamaño repositorio, decenas de minutos, y ahora con 57 mutantes más que al empezar |
| Un repositorio en cadena donde la clausura es todo el módulo | La ronda 016 midió los dos extremos de una cadena, no un repositorio real |
| Fuentes sin `gofmt` | **No obtienen gating alguno.** `schemata.Plan` rechaza una diferencia que arrastra formato. Se reporta como `none`, así que es visible — pero es un acantilado |

## 10. La afirmación que estos números sostienen

> En un módulo de tres paquetes con suite ligera, `--gated` produce los mismos veredictos y las mismas direcciones de supervivientes en aproximadamente un tercio del tiempo de pared; en uno de diez paquetes, en aproximadamente dos quintos, después de la clausura de observabilidad.

Todo lo que va más allá de eso —"ditto es N veces más rápido"— no está medido y no se afirma.

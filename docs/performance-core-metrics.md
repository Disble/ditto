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
| `mutantsPerReleaseOnThisRepository` | 789 | **873** |

Los ocho primeros **no se movieron**, y eso es el resultado: la ruta optimizada es opt-in y no toca el camino que esos contadores miden.

El noveno subió, y cada salto está atribuido por archivo, no al cambio entero:

| Salto | Causa | Archivos |
|---|---:|---|
| 789 → 813 (+24) | núcleo de alcance modular | `module_scope.go` +23, `gatedlaboratory.go` +1, `options.go` +0 |
| 813 → 818 (+5) | el motivo del veredicto en la ruta modular | `module_scope.go` 23 → 28 |
| 818 → 846 (+28) | la clausura de observabilidad | `module_scope.go` 28 → 54, `gatedlaboratory.go` 37 → 39 |
| 846 → 850 (+4) | un sandbox y un directorio de compilación por release | `gatedlaboratory.go` 39 → 42, `module_scope.go` 54 → 55 |
| 850 → 873 (+23) | lotes de compilación por nombre de binario (la colisión, entrada 022 del log) | `module_scope.go` 55 → 78 — el 78 es aritmética (55 registrado + 23 medidos), no un conteo por archivo, porque no existe tal contador. Un estado anterior del mismo cambio contó 874; la extracción que dejó `prepare` dentro del límite `cyclop` del gate movió el conteo en uno, y el número anotado es el que el gate contó sobre el árbol commiteado |

Cada suma coincide con lo que reportó el ratchet.

## 2. El mecanismo, aislado

Fixture de 3 paquetes y 12 mutantes, que ejecuta **4 binarios de prueba de paquete**. Contadores exactos, sin tiempo de pared. Corregido el 2026-09-18: esta sección llamaba al fixture un módulo de tres paquetes, pero lo que se ejecuta son cuatro binarios de prueba de paquete, y eso es lo que produce las 52 ejecuciones de la tabla.

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
| El gate completo de este repositorio | El tiempo de pared a tamaño repositorio sigue sin medirse; la muestra acotada de tres mutantes de §12 sólo prueba engagement y fidelidad |
| Un repositorio en cadena donde la clausura es todo el módulo | La ronda 016 midió los dos extremos de una cadena, no un repositorio real |
| Fuentes sin `gofmt` | **No obtienen gating alguno.** `schemata.Plan` rechaza una diferencia que arrastra formato. Se reporta como `none`, así que es visible — pero es un acantilado |

## 10. La afirmación que estos números sostienen

> En un módulo de tres paquetes con suite ligera, `--gated` produce los mismos veredictos y las mismas direcciones de supervivientes en aproximadamente un tercio del tiempo de pared; en uno de diez paquetes, en aproximadamente 0,22–0,23 del tiempo de pared, después de la clausura de observabilidad y de un sandbox y un directorio de compilación por release.

Corregido el 2026-09-18: esta afirmación cerraba con «aproximadamente dos quintos», que era el número de la clausura sola (0,4005–0,4040). La sección 4 ya registra 0,2230–0,2312 después del sandbox y del directorio de compilación compartidos, así que la afirmación ahora lleva ese número y su condición.

Todo lo que va más allá de eso —"ditto es N veces más rápido"— no está medido y no se afirma.

## 11. La colisión de nombres de binarios, encontrada y arreglada (2026-09-18)

La ruta modular nombraba los binarios de prueba sólo desde `path.Base(importPath)`, así que en ditto mismo `github.com/Disble/ditto`, `.../cmd/ditto` y `.../internal/ditto` producían todos `ditto.test.exe` (`dittotesting` colisionaba dos veces). El runner rehusaba el módulo entero (`Built=false`, `Compilations=0`, `PackageRuns=0`, `module test binary name collision`), y el gating modular sobre ditto gateó **0 de 850 mutantes** aunque `schemata` puede expresar 517 de ellos (60,8%), medido por `internal/perfbench/gating_test.go` sobre una copia sin `.git` de `85f2ea7`.

El arreglo agrupa los paquetes en lotes de compilación por nombre de binario (case-folded en Windows), cada lote escribe en su propio `batch-N` bajo el directorio de salida del release, y la unión de los argumentos de los lotes es el alcance descubierto — porque `go list -deps -test -json ./...` no hace type-check, y un argumento que no compile falla cerrado en vez de pasar en silencio.

Control y evidencia a través del binario, sobre un módulo desechable con colisión (dos paquetes que ambos producen `pay.test.exe`, 21 mutantes, `--threshold 0`):

| Momento | Total / killed / survived | Línea `Gated:` |
|---|---:|---|
| Binario anterior, mismo fixture | 21 / 15 / 6 | `none of 21 mutants ran from one compilation; 21 kept their own.` |
| Binario con el arreglo | 21 / 15 / 6 | `12 of 21 mutants ran from one compilation; 9 kept their own.` |

Las seis direcciones de supervivientes ordenadas son byte a byte iguales entre las dos rutas (sha256 `c1957c66a5268e8a2d6d52667abdd353158648d4ce134ea6d74918f4ddb7e8ec`), y las pruebas de los dos paquetes se probaron corriendo desde `batch-0/pay.test.exe` y `batch-1/pay.test.exe`.

Costo: un módulo con colisión paga una invocación de compilación por lote en colisión (en ditto son tres lotes); un módulo sin colisión sigue pagando exactamente una. El ratchet se movió 850 → 873 (+23), todo en `module_scope.go` (sección 1).

**Lo que NO estaba medido en esta entrada:** la proporción gateada realizada sobre el propio árbol de ditto. Un release acotado sobre `internal/fstemporarydir` (20 mutantes contra una suite de ~70 s por re-ejecución) murió por su propio presupuesto de tiempo después de la línea base y antes de cualquier veredicto. Lo que sí quedó medido aquí es que la ruta modular ya compila: el runner real sobre una copia sin `.git` de este árbol reportó `Built=true`, `Discoveries=1`, `ToolchainStarts=4`, `Compilations=3`, `PackageRuns=45`, `SkippedPackages=0`, sin texto de error, en 94,3 s — contra `Built=false` y `Compilations=0` antes del arreglo. La sección 12 cierra después la pregunta de engagement para una muestra real de tres mutantes; no convierte esa muestra en una estimación de todo el árbol.

## 12. Engagement realizado en ditto: muestra acotada de tres mutantes (2026-09-18)

Binario construido desde un archivo completo sin `.git` de `5320273232f5993a06ff96c47887cffd5c8a032f` (árbol `dc85e4232f422e1eba41054577ffef2551a22177`, sha256 del binario `572ea5104f5355201608f84b54a1cfa9d8b896a2e3a7ec6aa37a644a7318edc5`). La mutación corrió sólo en un repositorio desechable autocontenido. Un comentario semánticamente neutro en `internal/schemata/gate.go:148` produjo el rango staged `4961-5031`.

El plan estático produjo exactamente tres mutantes: dos admitidos por schemata (selectores 1 y 2) y uno rechazado (selector 0). La medición por el binario real confirmó esa división:

| Camino | Generados | Puntuables | Killed | Survived | No viable | `Gated:` |
|---|---:|---:|---:|---:|---:|---|
| Ordinario | 3 | 2 | 1 | 1 | 1 | — |
| `--gated` | 3 | 2 | 1 | 1 | 1 | `2 of 3`; 1 conservó su camino |

Las identidades ordenadas de los tres mutantes son byte a byte iguales (sha256 `abd0fd8a6eb4c6a86133602d127c441355d19d6cb1870fcc20bbcd7b8a1e3592`). La comparación de supervivientes no es vacía: ambos caminos reportaron exactamente `internal/schemata/gate.go:148:22 → Comparison Replace (found != nil → true)`, con sha256 `4e16653a546ae42ce441c8272932cf612ec8b51d2171369e2fb9f15869294d6e`.

**Conclusión exacta:** el arreglo de colisiones sí permite que el gating se ejecute sobre el árbol real de ditto, y en esta muestra preserva veredictos, no viables, identidades y la dirección del superviviente. La proporción realizada de esta muestra es 2/3; **no** estima la proporción de los 873 mutantes actuales ni reemplaza el techo sintáctico anterior de 517/850.

Tiempo de pared, sólo reportado: 186 s ordinario contra 304 s gateado (razón 1,6344), con líneas base de 1m5,832s y 1m11,832s. No hubo rondas rotadas. La única lectura permitida es que tres lotes de compilación no se amortizan con sólo dos mutantes gateados; el punto de cruce sigue sin medirse.

## 13. El denominador real: fail-fast (2026-09-18)

Los gates reales no usan el comando exacto que admite la ruta modular: los tres gates de ditto pasan por `test.failfast`, y dharness usa un comando package-local. `scopeOf` sólo admite tres grafías del `go test ... ./...` por defecto; cualquier otro comando desactiva la optimización. Por eso las ganancias anteriores de 0,22–0,32 comparaban contra una suite completa por mutante, no contra el costo que esos gates pagan.

Se repitió el rango exacto de §12 con `go test -count=1 -json -failfast ./...`, una vez sin `--gated` y otra con él como control de alcance:

| Camino | Tiempo | Generados / puntuables / killed / survived / no viable | `Gated:` |
|---|---:|---:|---|
| Ordinario por defecto (§12) | 186 s | 3 / 2 / 1 / 1 / 1 | — |
| Fail-fast | 163 s | 3 / 2 / 1 / 1 / 1 | — |
| Fail-fast + `--gated` | 162 s | 3 / 2 / 1 / 1 / 1 | `none of 3`; 3 conservaron su camino |
| Ruta modular actual (§12) | 304 s | 3 / 2 / 1 / 1 / 1 | `2 of 3`; 1 conservó su camino |

Las identidades de mutantes y la línea no vacía del superviviente tienen los mismos hashes de §12 (`abd0fd8a…e3592` y `4e16653a…d6e`). El control `none of 3` confirma que fail-fast no se combinó accidentalmente con la ruta modular.

Regla fijada antes de medir: fail-fast debía tardar como máximo 93 s para ser por sí solo una mejora drástica; tardó 163 s. La ruta modular debía superar a fail-fast por al menos 30%; tardó 304 s, **86,5% más**. Ambas ramas aisladas quedan rechazadas.

**Decisión arquitectónica:** no seguir perfilando el camino modular contra el denominador equivocado. El siguiente candidato combina compilación compartida con semántica fail-fast: admitir esa forma configurada, pasar fail-fast a los binarios de prueba y detener la ejecución de paquetes observadores al primer kill. Sólo un prototipo desechable que preserve todos los observables y gane al menos 30% end-to-end contra fail-fast puede promover esa dirección. Estos tiempos son muestras únicas sin orden rotado; sirven para las líneas de decisión amplias anteriores, no como ratio publicable.

## 14. Compilación compartida + fail-fast: dirección cerrada (2026-09-18)

El prototipo de §13 se construyó únicamente en una copia sin `.git`. Admitió la forma fail-fast exacta, pasó `-test.failfast` a cada binario de pruebas y dejó de iniciar paquetes observadores después del primer fallo. El scope real fueron dos líneas staged de `internal/schemata/instrument.go`: exactamente 10 mutantes y 10 selectores gateables.

Todos los controles corrieron antes de medir: rechazo sin admisión (`none of 10`), RED deliberado sin fail-fast, GREEN con el flag, restauración de un package run al desactivar early stop, invariancia del modo normal, fallos cerrados de discovery/build/binario ausente/scope vacío/type error y rechazo de un contador falso 999 contra 1.

| Ronda | Orden | Fail-fast ordinario | Combinado | Combinado / ordinario |
|---|---|---:|---:|---:|
| warm-up descartado | ordinario → combinado | 622 s | 692 s | 1,1125 |
| 1 | ordinario → combinado | 623 s | 703 s | **1,1284** |
| 2 | combinado → ordinario | 662 s | 715 s | **1,0801** |
| 3 | ordinario → combinado | 676 s | 705 s | **1,0429** |

La fidelidad fue exacta en las seis corridas medidas: 10 total / 8 killed / 2 survived, hashes iguales para identidades (`19d03754…bf92`), dos supervivientes no vacíos (`4d9c3c8b…9ce26`) y razones ordenadas (`fbdda811…f5b4af`: 8 assertion, 2 unknown). Cada corrida combinada gateó 10 de 10 y terminó con los mismos contadores: 11 selecciones, 1 discovery, 4 arranques de toolchain, 3 compilaciones, 221 package runs, **208 paquetes detenidos**, 66 skips por clausura y 8 converters. El control causal exacto —un paquete detenido contra un package run restaurado sin mover el veredicto— se ejecutó en una fixture pequeña; el scope real confirma los 208 detenidos, pero no repitió sus 429 package runs potenciales con early stop desactivado.

La regla exigía razón ≤0,70 en las tres rondas. Dio **1,1284 · 1,0801 · 1,0429**. El prototipo quitó 208 arranques de paquetes y aun así fue más lento en todas las rondas.

**Decisión:** cerrar la optimización del camino modular como dirección principal. El contador de paquetes iniciados se movió drásticamente en la dirección esperada sin mover la latencia del usuario; es un proxy insuficiente para este workload. No se promueve código. La próxima hipótesis debe quitar ejecución de tests, no sólo driver o coordinación de paquetes, y volver a exigir fidelidad más una ganancia end-to-end de al menos 30% contra fail-fast ordinario.

## 15. El techo seguro: los tests son el 99,3% (2026-09-18)

Última medición de la rama, con la precisión como invariante: **cada mutante viable ejecuta completa la suite configurada**, sin omitir, muestrear ni priorizar tests. Se instrumentó únicamente el límite del proceso ya existente en `internal/cmdtestrunner`, en una copia descartable, con el mismo scope real de 10 mutantes y el mismo comando `go test -count=1 -json -failfast ./...`.

Cada corrida produjo exactamente once invocaciones del comando configurado (1–11, todas positivas): tres pasaron —baseline verde y los dos supervivientes— y ocho fallaron, correspondiendo a los ocho kills.

| Corrida | Dentro del comando | End-to-end | Proporción | Cota superior no-test |
|---|---:|---:|---:|---:|
| warm-up descartado | 621,148 s | 623,400 s | 0,996388 | 2,252 s |
| 1 | 618,717 s | 623,011 s | **0,993109** | 4,293 s |
| 2 | 644,116 s | 648,537 s | **0,993183** | 4,421 s |
| 3 | 601,268 s | 605,427 s | **0,993131** | 4,159 s |

La fidelidad se mantuvo: 10 total / 8 killed / 2 survived, score 0,80, con los mismos hashes de §14 (identidades `19d03754…bf92`, supervivientes `4d9c3c8b…9ce26`, razones `fbdda811…f5b4af`: 8 assertion, 2 unknown).

El residuo de 2,252–4,421 s es una **cota superior de todo lo que no es el comando configurado**: incluye arranque y cierre de procesos, lecturas de reloj del shell, staging previo, reporte posterior y ruido de la máquina. **No** es una medición del overhead propio de ditto y nada acá atribuye esos segundos a un componente. El instrumento tampoco identifica cada registro con un mutante: la correspondencia 1:1 se apoya en el contador monotónico y en la composición 10/8/2.

**Conclusión de la rama:** la regla pre-registrada se cumplió —99,3% en las tres rondas válidas— así que **no queda rango de mejora medido que preserve la precisión**. Lo único que habría dado una ganancia drástica por esta vía era ejecutar menos tests, y eso se rechazó explícitamente porque cambia la precisión. La rama se cierra por hoy; no se promueve código.

Límites: un scope staged, diez mutantes, un archivo, un repositorio, una máquina y un sistema operativo. No establece la misma proporción a tamaño repositorio, con suite pesada, en otras plataformas, ni para el gate completo.

## 16. Paralelismo exterior explícito: factible, pero sin ganancia suficiente (2026-09-19)

Se evaluó primero `Parallel()` sin cambiar su mecanismo. Aunque el host pidió dos lanes, el comando configurado nunca superó una invocación activa: el camino verbose espera el lote interno antes de que continúen los subtests paralelos de reporte. Ese mecanismo quedó refutado como base de medición.

Un segundo prototipo, pre-registrado por separado y construido sólo en una copia descartable, usó futuros indexados y un límite explícito alrededor del laboratorio ordinario. Los controles dieron RED a máximo 1 antes, GREEN a máximo 2 después, fallo al mutar manualmente la capacidad otra vez a 1 y conservación del orden de resultados bajo finalización deliberadamente desordenada.

El scope real fue `internal/schemata/gate.go:148`: 3 generados / 2 scored / 1 killed / 1 survived / 1 non-viable. Todos los brazos reprodujeron la composición, el superviviente y las razones (sha256 `6c9f12c7…881f874`), con baseline verde y cuatro starts/ends balanceados.

| Corrida | Workers | Máximo activo | End-to-end | Mínimo disponible | Pico Job Object |
|---|---:|---:|---:|---:|---:|
| warm-up descartado | 1 | 1 | 165,707 s | 15.351.324.672 B | 4.377.591.808 B |
| warm-up descartado | 2 | 2 | 156,118 s | 16.199.344.128 B | 3.422.113.792 B |
| ronda válida 1 | 1 | 1 | 163,748 s | 15.137.050.624 B | 3.655.745.536 B |
| ronda válida 1 | 2 | 2 | 154,543 s | 17.354.833.920 B | 3.757.441.024 B |

El ratio válido fue **0,943784**, apenas **5,62%** menos tiempo, frente al requisito pre-registrado `<=0,75`. La regla mataba H2 con cualquier ronda sobre ese límite, así que no se gastaron dos pares adicionales ni se probó un tercer worker. La admisión adaptativa dependía de H2 y tampoco se construyó.

Los tres comandos mutantes duraron 2,711 / 27,309 / 64,069 s en serial y 12,727 / 50,439 / 72,611 s con solapamiento. Eso explica aritméticamente por qué el outer wall-clock apenas bajó; no identifica la causa de la contención.

**Decisión:** no promover scheduler ni controlador de memoria. Go puede implementar correctamente el mecanismo y Windows mostró amplio headroom, pero el lever medido no pagó su complejidad. Reabrir requiere otro workload nombrado y una hipótesis que pueda superar este resultado sin reducir tests ni cambiar veredictos.

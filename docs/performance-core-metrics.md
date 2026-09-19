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
| El gate propio de este repositorio | Tamaño repositorio, decenas de minutos, y ahora con 57 mutantes más que al empezar |
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

**Lo que NO está medido:** la proporción gateada realizada sobre el propio árbol de ditto. Un release acotado sobre `internal/fstemporarydir` (20 mutantes contra una suite de ~70 s por re-ejecución) murió por su propio presupuesto de tiempo después de la línea base y antes de cualquier veredicto, así que no se leyó ninguna línea `Gated:` para el árbol de ditto. Lo que sí está medido en ditto mismo es que la ruta modular ya compila: el runner real sobre una copia sin `.git` de este árbol reporta `Built=true`, `Discoveries=1`, `ToolchainStarts=4`, `Compilations=3`, `PackageRuns=45`, `SkippedPackages=0`, sin texto de error, en 94,3 s — contra `Built=false` y `Compilations=0` antes del arreglo. El techo sintáctico es 517 de 850; lo realizado queda sin medir.

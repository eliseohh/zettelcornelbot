# ZettelCornelBot: Instrucciones de Migración (Apple Silicon)

> [!IMPORTANT]
> Este documento es exclusivo para el despliegue en arquitectura **macOS M4+ (ARM64)**.
> El proyecto ha sido incubado en un entorno Linux estricto. Al migrar, se deben activar los módulos latentes de alto rendimiento.

## 1. Validación de Entorno

Asegurar stack nativo:
```bash
sysctl -n machdep.cpu.brand_string  # Debe contener "Apple M4" o superior
swift --version                     # Requerido Swift 6.0+
go version                          # Requerido Go 1.22+ (darwin/arm64)
```

## 1.1 Inicio Rápido (Setup)

```bash
# 1. Clonar repositorio
git clone https://github.com/eliseohh/zettelcornelbot.git
cd zettelcornelbot

# 2. Verificar integridad y levantar servicios
make run
```

## 1.2 Verificación de Arranque

Al ejecutar `make run`, el sistema debe reportar:

```text
✅ CUMPLIMIENTO VERIFICADO
✅ Ollama is RUNNING
✅ Telegram Token found
PASS: TestBotHandlers
🚀 Starting zettelbot...
```

Si algún paso falla, corregir antes de intentar la migración a Metal.

## 2. Activación de Módulos Latentes

Actualmente, los siguientes módulos son **STUBS** (marcadores de posición) en Linux. Deben implementarse nativamente en macOS.

### A. Visualización (Metal)
Ruta: `internal/visual/renderer.go`
- **Estado Actual**: Devuelve error "not implemented".
- **Objetivo M4**:
  - Implementar binding CGO hacia Swift/Metal.
  - Usar `MTKView` y shaders de cómputo para renderizar el grafo de notas (nodos + aristas) directamente en GPU.
  - Referencia: Usar buffers compartidos entre Go (DB) y Metal para evitar copias.

### B. Inferencia Neuronal (CoreML / MPS)
Ruta: `internal/neural/execute.go` (a crear)
- **Estado Actual**: Cliente HTTP hacia Ollama.
- **Objetivo M4**:
  - Permitir inferencia local acelerada.
  - Opción 1: Compilar `llama.cpp` con soporte Metal explícito (`LLAMA_METAL=1`).
  - Opción 2: Usar bindings directos a CoreML para embeddings ultra-rápidos de las notas al indexar.

## 3. Protocolo de Continuidad

A pesar del cambio de hardware, los **Principios Duros** se mantienen INMUTABLES:

1.  **Markdown es la Verdad**: NUNCA usar CoreData o SQLite para almacenamiento semántico.
2.  **Bot Estricto**: No agregar conversaciones "blandas" solo porque el modelo sea más rápido.
3.  **Auditoría**: `audit.sh` y `cmd/compliance` deben seguir ejecutándose antes de cada commit. Si Metal rompe el compliance, se descarta Metal, no el compliance.

## 4. Pasos de Activación

1.  Clonar repo en Mac.
2.  Ejecutar `./audit.sh` para verificar integridad base.
3.  Crear rama `feature/metal-render`.
4.  Implementar `internal/visual/metal_bridge.m` (Objective-C wrapper).
5.  Reemplazar stub en `internal/visual`.

🚀 **Objetivo Final**: Sistema de cognición local con latencia cero y renderizado de grafo en tiempo real a 120Hz (ProMotion).



## 5. LLM Continuity Contract

Antigravity debe asumir:

- Ningún cambio de hardware habilita cambios semánticos.
- Metal, CoreML o MPS NO alteran:
  - estructura de notas
  - límites cognitivos
  - flujo de validación
- Toda optimización es reversible.
- Si existe duda entre:
  - performance vs compliance
  → elegir compliance.

Antigravity debe continuar desde este documento
como fuente de verdad operativa.

## 7. Inferencia Acelerada (Contrato)

CoreML / MPS / Metal inference:
- Entrada: texto explícito
- Salida: texto temporal
- Persistencia: prohibida

Embeddings:
- Solo en memoria
- Nunca en disco
- Nunca como fuente de verdad

La inferencia es una función pura.

## 9. Referencia de Implementación (Swift/Metal)

Copiar estos archivos en `internal/visual/macos/` al migrar.

### A. Tipos Compartidos (`SharedTypes.h`)
Define la estructura de datos que Go enviará a C/Swift.
```c
#ifndef SharedTypes_h
#define SharedTypes_h

#include <simd/simd.h>

typedef struct {
    vector_float2 position;
    vector_float4 color; // Mapeado desde 'Tipo' (Libro=Azul, Idea=Amarillo, etc)
    float size;
    // int typeID; // Opcional para filtrado en shader
} NodeVertex;

typedef struct {
    matrix_float4x4 viewProjectionMatrix;
} Uniforms;

#endif /* SharedTypes_h */
```

### B. Shader Metal (`Shaders.metal`)
Renderiza puntos (nodos) eficientemente.
```metal
#include <metal_stdlib>
#include "SharedTypes.h"

using namespace metal;

struct VertexOut {
    float4 position [[position]];
    float4 color;
    float size [[point_size]];
};

vertex VertexOut vertex_main(const device NodeVertex *vertices [[buffer(0)]],
                             constant Uniforms &uniforms [[buffer(1)]],
                             uint vertexID [[vertex_id]]) {
    VertexOut out;
    NodeVertex v = vertices[vertexID];
    
    out.position = uniforms.viewProjectionMatrix * float4(v.position, 0.0, 1.0);
    out.color = v.color;
    out.size = v.size;
    
    return out;
}

fragment float4 fragment_main(VertexOut in [[stage_in]]) {
    // Dibujar círculo suave
    float2 coord = in.position.xy; 
    // (Lógica de suavizado de punto aquí...)
    return in.color;
}
```

### C. Renderer Swift (`Renderer.swift`)
Puente que inicializa MTKView y gestiona el pipeline.
```swift
import MetalKit

class Renderer: NSObject, MTKViewDelegate {
    var device: MTLDevice!
    var commandQueue: MTLCommandQueue!
    var pipelineState: MTLRenderPipelineState!
    var vertexBuffer: MTLBuffer?
    
    init(metalView: MTKView) {
        super.init()
        device = metalView.device
        commandQueue = device.makeCommandQueue()
        buildPipeline(view: metalView)
    }
    
    func buildPipeline(view: MTKView) {
        let library = device.makeDefaultLibrary()!
        let header = MTLRenderPipelineDescriptor()
        header.vertexFunction = library.makeFunction(name: "vertex_main")
        header.fragmentFunction = library.makeFunction(name: "fragment_main")
        header.colorAttachments[0].pixelFormat = view.colorPixelFormat
        
        pipelineState = try! device.makeRenderPipelineState(descriptor: header)
    }
    
    func updateData(nodes: UnsafeRawPointer, count: Int) {
        // Copia eficiente de memoria Go -> Swift -> GPU
        // O mejor: Go -> C Buffer -> GPU (Zero Copy)
        let size = count * MemoryLayout<NodeVertex>.stride
        vertexBuffer = device.makeBuffer(bytes: nodes, length: size, options: .storageModeShared)
    }
    
    func draw(in view: MTKView) {
        guard let buffer = vertexBuffer,
              let descriptor = view.currentRenderPassDescriptor,
              let encoder = commandQueue.makeCommandBuffer()?.makeRenderCommandEncoder(descriptor: descriptor) 
        else { return }
        
        encoder.setRenderPipelineState(pipelineState)
        encoder.setVertexBuffer(buffer, offset: 0, index: 0)
        encoder.drawPrimitives(type: .point, vertexStart: 0, vertexCount: buffer.length / MemoryLayout<NodeVertex>.stride)
        encoder.endEncoding()
        encoder.commandBuffer.present(view.currentDrawable!)
        encoder.commandBuffer.commit()
    }
    
    func mtkView(_ view: MTKView, drawableSizeWillChange size: CGSize) {}
}
```

### D. Puente Go (CGO)
Desde Go, llamarás a funciones C exportadas que invoquen este código Swift.
Requerirá bandera `-framework Metal -framework MetalKit`.


Cuando Antigravity retome el proyecto debe:

1. Leer este documento completo
2. Validar principios duros
3. Identificar stubs activos
4. Proponer implementación SIN alterar SPEC
5. Declarar explícitamente:
   - qué optimiza
   - qué no toca

## 10. Roadmap: TDAH Active Focus (v1.1)

El siguiente paso NO es solo rendimiento gráfico, sino **Cognición Activa**.
El usuario requiere un sistema que **inicie la interacción**.

### Especificación Funcional
1.  **Scheduler Loop**:
    - Goroutine en `cmd/bot/main.go`.
    - Ticker: 30 minutos (ajustable por `/config`).
2.  **Modos de Operación**:
    - **Work Mode**: "¿Sigues enfocado?" -> Log en Daily Note.
    - **Rest Mode**: Sugerencia de Ideas (Graph Rot).
3.  **Persistencia**:
    - Tabla `user_state` (singleton) para guardar `last_ping`, `current_task`, `work_mode_enabled`.

⚠️ **Prioridad**: Implementar Scheduler LIGERO en Go antes de meter toda la complejidad gráfica de Metal.
La lógica de negocio TDAH tiene precedencia sobre la visualización.


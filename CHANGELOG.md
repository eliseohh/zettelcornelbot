# CHANGELOG & Contexto Evolutivo

Este documento sirve como bitácora de contexto para instancias de IA futuras.

## [v1.1 - Planned] - "The Active Focus Update"
**Contexto**: El usuario requiere soporte activo para TDAH. El sistema pasivo (Zettelkasten puro) es insuficiente.
**Objetivo**: Transformar al bot en un "Accountability Partner".

### Roadmap Técnico
1.  **Scheduler (Goroutine)**: Loop infinito que despierta cada 30m.
2.  **Daily Log (`/daily`)**: Buffer efímero de pensamientos. Se purga al final del día.
3.  **Active Recall (`/review`)**: Sistema SRS para revivir notas viejas.
4.  **Work Mode (`/work`)**:
    - **ON**: Pings cada 30m ("¿Sigues en task X?").
    - **OFF**: Silencio o recordatorios suaves de ideas.

---

## [v1.0.1] - 2024-02-02
- **Feat (Bot)**: Implementación de Carpetas (`/note create libro Title`).
- **Feat (Bot)**: Status Tree View para visualizar categorías.
- **Perf (Indexer)**: Worker Pool (4 hilos) para hashing/parsing paralelo.
- **Docs**: `README_M4` con instrucciones de migración Apple Silicon.

## [v1.0.0] - 2024-02-02
- **Core**: Markdown Parser Inmutable (Strict Specifications).
- **Core**: Indexer SQLite con derivadas de ID por nombre de archivo.
- **Bot**: Comandos atómicos (`/note`, `/cue`, `/ai`).
- **Safety**: Script de Compliance (`audit.sh`) obligatorio.

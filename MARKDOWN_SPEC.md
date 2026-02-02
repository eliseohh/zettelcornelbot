# Protocolo de Verdad: Markdown Spec (INMUTABLE)

> [!IMPORTANT]
> Este formato es ESTRICTO. El parser rechazará o marcará como inválidos los archivos que no cumplan esta estructura o excedan los límites.

## Estructura Base

```markdown
# {Título}
Fecha: YYYY-MM-DD
Tipo: idea | estudio | libro | tarea

## Notas
{Contenido principal ~70%}

## Cues
- {Pregunta o clave de recuerdo?}

## Resumen
{Síntesis de la nota}

## Enlaces
- [[{ID}]]
```

## Reglas de Parseo y Límites

1. **Global**:
   - Longitud Total ≤ 4000 caracteres.

2. **Título**:
   - Línea 1, H1.
   - Longitud ≤ 120 caracteres.

3. **Metadatos**:
   - `Fecha` y `Tipo` obligatorios inmediatamente después del título.

4. **Secciones (Límites)**:
   - **Notas**: ≤ 2800 caracteres.
   - **Resumen**: ≤ 500 caracteres.

5. **Cues (Active Recall)**:
   - Máximo 7 items.
   - Cada item ≤ 120 caracteres.
   - **Debe terminar estrictamente con '?'**.

6. **Enlaces**:
   - Lista explícita bajo `## Enlaces`.

Cualquier violación a estos límites provocará un fallo de validación.

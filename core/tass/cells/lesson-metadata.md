# Junction cell: lesson metadata

This cell describes one information junction. It is data for the Core's fixed Go handler, not code:
nothing in this file runs, and it grants no permission (Identity grant `junction.read:lesson-metadata`).

id: junction.lesson-metadata
handler: lesson-metadata.v1
capability: READ_LESSON_METADATA
input: one lesson path that is on TASS's Learning card
fields: title, goal, checks, status
max_text: 300
purpose: Before I think about a lesson, I need its title, goal and checks.
expect: one line per field, or UNAVAILABLE when the lesson is not on the Learning card

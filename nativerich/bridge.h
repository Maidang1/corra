#include <stdint.h>
#include <stdlib.h>

uintptr_t cr_new(uintptr_t window, uintptr_t callback);
void cr_close(uintptr_t editor);
void cr_load(uintptr_t editor, const char *json);
char *cr_snapshot(uintptr_t editor, int *marked);
void cr_format(uintptr_t editor, const char *command);
void cr_insert(uintptr_t editor, const char *json);
void cr_frame(uintptr_t editor, double x, double y, double w, double h);
void cr_visible(uintptr_t editor, int visible);
void cr_style(uintptr_t editor, double zoom, int dark, int editable);
void cr_base_dir(uintptr_t editor, const char *path);
char *cr_render_math(char *source, int dark, double zoom, int inlineMath);
char *cr_render_block(char *language, char *body, int dark, double zoom);
void cr_focus(uintptr_t editor, int focus);
void cr_callback(uintptr_t handle, int source);
void cr_spellcheck(uintptr_t editor, int enabled);
void cr_focus_mode(uintptr_t editor, int enabled);
void cr_typewriter(uintptr_t editor, int enabled);
void cr_find(uintptr_t editor);
void cr_reveal_heading(uintptr_t editor, int index);
void cr_print(uintptr_t editor);

int cr_test_loop(void);
int cr_test_finish_event(uintptr_t editor);
void cr_test_exit(int code);
void cr_test_main(uintptr_t handle);
void cr_test_callback(uintptr_t handle);
uintptr_t cr_test_window(void);
void cr_test_close_window(uintptr_t window);
void cr_test_select(uintptr_t editor, int start, int length);
void cr_test_selection(uintptr_t editor, int *start, int *length);
void cr_test_action(uintptr_t editor, const char *action, const char *text);
int cr_test_traits(uintptr_t editor, const char *trait);
int cr_test_checkbox(uintptr_t editor);
void cr_test_opaque(uintptr_t editor, int whole);
int cr_test_screenshot(uintptr_t editor, const char *path);
int cr_test_layout(uintptr_t editor);
char *cr_test_text(uintptr_t editor);
char *cr_test_code_controls(uintptr_t editor);
char *cr_test_footnotes(uintptr_t editor);
char *cr_test_images(uintptr_t editor);
char *cr_test_toc(uintptr_t editor);
char *cr_test_clipboard_text(void);

char *cr_markdown_json(char *json);
char *cr_parse_markdown(char *source);
char *cr_parse_inline(char *source, int table);
char *cr_reveal_inline(char *runs, int table);
char *cr_fold_inline(char *source, int table);
char *cr_highlight(char *code, char *language);
char *cr_test_markers(uintptr_t editor);

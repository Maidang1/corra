//go:build darwin && cgo

#import <AppKit/AppKit.h>
#import <UniformTypeIdentifiers/UniformTypeIdentifiers.h>
#import "bridge.h"

// NSFilenamesPboardType is still how Finder file lists arrive.
#pragma clang diagnostic ignored "-Wdeprecated-declarations"

static NSString *const CRParagraph = @"CorraParagraph";
static NSString *const CRBold = @"CorraBold";
static NSString *const CRItalic = @"CorraItalic";
static NSString *const CRCode = @"CorraCode";
static NSString *const CRTableCell = @"CorraTableCell";
static NSString *const CRMathText = @"CorraMathText";
static NSString *const CRMathExpr = @"CorraMathExpr";
static NSString *const CRMathNumber = @"CorraMathNumber";
static NSString *const CRMathRendered = @"CorraMathRendered";
static NSString *const CRInlineSource = @"CorraInlineSource";
static NSString *const CRLinkTitle = @"CorraLinkTitle";
static NSString *const CRUnderline = @"CorraUnderline";
static NSString *const CRKbd = @"CorraKbd"; // <kbd>key</kbd>
static NSString *const CRHighlight = @"CorraHighlight";
static NSString *const CRSup = @"CorraSup";
static NSString *const CRSub = @"CorraSub";
// CRColor/CRBackground hold the stored CSS color of <span style="color:…">
// and background-color; drawing resolves them per theme.
static NSString *const CRColor = @"CorraColor";
static NSString *const CRBackground = @"CorraBackground";
// CRMention marks a page mention: a "[@Title](note.md)" link drawn as a chip
// that opens the note on click and deletes as one unit.
static NSString *const CRMention = @"CorraMention";
static NSString *const CRHl = @"CorraHl"; // syntax-highlight token class on code
static NSString *const CRSoftBreak = @"CorraSoftBreak"; // a U+2028 that was a plain source newline
static NSString *const CRFootnote = @"CorraFootnote";
// CRLiteral marks punctuation typed as a Markdown escape (`\*`): it is text,
// never a delimiter for the inline rules.
static NSString *const CRLiteral = @"CorraLiteral";
// CRHTML marks a raw inline HTML tag kept as source (`<span>`, `<!-- -->`).
static NSString *const CRHTML = @"CorraHTML";
// CRImageSource/CRImageURL mark an inline image attachment with its exact
// ![alt](src) source and destination.
static NSString *const CRImageSource = @"CorraImageSource";
static NSString *const CRImageURL = @"CorraImageURL";
static NSString *const CRFootnoteNumber = @"CorraFootnoteNumber";
static NSString *const CRFootnoteFigureSource = @"CorraFootnoteFigureSource";
static NSString *const CROwnPaste = @"app.corra.rich-document";
// Go-side handlers for parity features (cgo exports, main thread only).
extern char *cr_emoji(const char *word);
extern char *cr_emoji_search(const char *query);
extern char *cr_link_candidates(uintptr_t handle, const char *query);
extern int cr_open_document(uintptr_t handle, const char *target);
extern char *cr_page_info(const char *path);
extern char *cr_paste_image(uintptr_t handle, const char *pngBase64, const char *name);
extern char *cr_drop_files(uintptr_t handle, const char *pathsJSON);
extern char *cr_html_to_markdown(const char *html);
extern char *cr_render_math_document(uintptr_t handle, const char *document, int dark, double zoom, const char *numbering);
@class CREditor;
@class CRSlashMenuView;
@class CRFormatBarView;
@class CRFootnoteCardView;
@class CREmojiMenuView;
@class CRLinkMenuView;
@interface CRTextView : NSTextView {
    NSRange _hoverCodeBlock;
    NSRect _hoverCodeChip;
    NSRect _copyRect;
    NSRange _hoverBlock;
    NSRect _handleRect;
    NSPoint _hoverPoint;
    BOOL _copied;
    NSUInteger _copyToken;
    NSRange _dragSource;
    NSInteger _dragInsertion;
    BOOL _dragActive;
    NSRange _languageRange;
    NSRect _languageRect;
    NSRange _hoverImage;
    NSRect _imagePill;
    NSArray *_imageButtons;
    NSRect _imageReplaceRect;
    NSRect _imageCopyRect;
    NSRect _imageDeleteRect;
    NSPanel *_footnotePanel;
    CRFootnoteCardView *_footnoteCard;
    NSString *_footnoteLabel;
    NSRect _footnoteRect;
    NSString *_hoverFootnote;
    NSUInteger _hoverFootnoteLocation;
    NSTextTable *_hoverTable;
    NSRect _tableRect;
    NSArray *_tableButtons;
    // Table row/column drag: grips on the hovered row's left edge and the
    // column's top edge, a live insertion line, and the dragged band.
    NSInteger _tableDragRow;      // body row being dragged (0 = header, -1 = none)
    NSInteger _tableDragColumn;   // column being dragged (-1 = none)
    NSInteger _tableDragTarget;   // insertion slot (0...count)
    BOOL _tableDragActive;
    BOOL _tableDragIsRow;
    NSRect _tableDragRowGrip;     // left-edge grip of the hovered row
    NSRect _tableDragColumnGrip;  // top-edge grip of the hovered column
    NSInteger _tableHoverRow;     // row under the pointer (-1 = none)
    NSInteger _tableHoverColumn;  // column under the pointer (-1 = none)
    NSTrackingArea *_hoverTracking;
}
@property(nonatomic, weak) CREditor *owner;
- (void)clearHover;
- (void)clearDrag;
- (BOOL)copyImageAtHover;
- (void)replaceImageAtHover;
- (BOOL)deleteImageAtHover;
- (NSRect)imagePillButtonRect:(NSInteger)index;
- (NSRect)tableButtonRectForKind:(NSInteger)kind;
- (void)hideFootnoteCard;
- (void)closeFootnoteCard;
- (void)footnoteCardJump;
- (NSDictionary *)footnoteCardState;
- (BOOL)footnoteCardSnapshotTo:(NSString *)path;
- (void)pickCodeLanguage:(NSMenuItem *)item;
@end
@interface CREditor : NSObject <NSTextViewDelegate> {
    NSPanel *_slashPanel;
    CRSlashMenuView *_slashView;
    NSArray *_slashItems;
    NSString *_slashFilter;
    NSInteger _slashSelected;
    NSPanel *_emojiPanel;
    CREmojiMenuView *_emojiView;
    NSString *_emojiFilter;
    NSInteger _emojiSelected;
    NSRange _emojiQuery;
    NSPanel *_linkPanel;
    CRLinkMenuView *_linkView;
    NSString *_linkFilter;
    NSInteger _linkSelected;
    NSRange _linkQuery;
    BOOL _linkMention;      // the open menu answers an "@" mention, not "[["
    BOOL _linkCard;         // ...and the pick becomes a page card (slash menu)
    NSUInteger _linkCardAt; // where that card's "@" was typed
    NSPanel *_formatPanel;
    CRFormatBarView *_formatView;
    NSUInteger _footnoteBack; // caret spot the last footnote jump started from; NSNotFound when none
}
@property NSScrollView *scroll;
@property CRTextView *text;
@property NSUndoManager *undo;
@property(nonatomic, weak) NSWindow *window;
@property(nonatomic, weak) NSResponder *previousResponder;
@property uintptr_t callback;
@property BOOL suppress;
@property BOOL dark;
@property double zoom;
@property NSString *bodyFont; // settings' body family; nil = system font
@property NSString *monoFont; // settings' code family; nil = system mono
@property NSDictionary *emptyParagraph;
@property NSString *baseDir;
@property NSMutableDictionary *mathCache;
@property NSString *mathNumbering;
@property NSString *mathJSON;
@property NSArray *mathResults;
@property NSString *mathSignature; // math inputs at the last refresh; nil forces one
@property NSArray *preparedMath;
@property NSUInteger mathIndex;
@property NSMutableDictionary *imageCache;
@property NSMutableDictionary *imagePending;
@property NSMutableDictionary *imageAttachments;
@property NSMutableDictionary *imageSpecForURL; // the size suffix a URL was fetched for
@property NSString *tocSignature; // the heading list the [TOC] figure was drawn from
@property NSRange ruleCaret; // paragraphs the caret was in at the last rule refresh
@property NSRange dirty;
@property NSDictionary *mergeParagraph; // block a cross-paragraph deletion started in
@property NSUInteger mergeLocation; // paragraphs whose styling an edit may have changed; NSNotFound when clean
@property NSRange draftGroup; // contiguous source/body paragraphs the caret is in
@property BOOL focusMode;
@property BOOL typewriter;
@property BOOL codeLineNumbers;
@property CGFloat codeGutter;
@property NSUInteger documentRevision;
@property NSUInteger lastCaret;
@property NSDictionary *footnotes;
- (NSArray *)footnoteRuns:(NSArray *)runs;
- (void)prepareFootnotes:(NSDictionary *)document;
- (NSAttributedString *)footnote:(NSDictionary *)run attributes:(NSDictionary *)attrs;
- (void)refreshFootnotes;
- (void)navigateFootnote:(BOOL)definition atLocation:(NSUInteger)location;
- (void)clearFootnoteBack;
- (BOOL)footnoteCanNavigateBack;
- (BOOL)footnoteNavigateBack;
- (void)insertFootnote;
- (NSString *)footnoteLabelAt:(NSUInteger)location;
- (BOOL)allowFootnoteRange:(NSRange)range;
- (NSArray *)footnoteFragment:(NSArray *)paragraphs container:(NSDictionary *)container;
- (void)restyle;
- (void)normalize;
- (void)notify;
- (void)source;
- (BOOL)allowRange:(NSRange)range;
- (void)format:(NSString *)command;
- (void)slashQueryChanged;
- (BOOL)slashHandleKey:(NSEvent *)event;
- (void)slashClose;
- (void)slashChoose:(NSDictionary *)item;
- (void)slashChooseIndex:(NSInteger)index;
- (void)emojiMenuQueryChanged;
- (BOOL)emojiMenuHandleKey:(NSEvent *)event;
- (void)emojiMenuClose;
- (void)emojiMenuChoose:(NSDictionary *)item;
- (void)emojiMenuChooseIndex:(NSInteger)index;
- (void)emojiMenuParentChanged:(NSNotification *)note;
- (NSString *)emojiMenuFilterText;
- (NSDictionary *)emojiMenuState;
- (void)linkMenuQueryChanged;
- (BOOL)linkMenuHandleKey:(NSEvent *)event;
- (void)linkMenuClose;
- (void)linkMenuChoose:(NSDictionary *)item;
- (void)linkMenuChooseIndex:(NSInteger)index;
- (void)linkMenuParentChanged:(NSNotification *)note;
- (NSString *)linkMenuFilterText;
- (BOOL)linkMenuHeadingMode;
- (NSDictionary *)linkMenuState;
- (BOOL)emojiMenuSnapshotTo:(NSString *)path;
- (NSDictionary *)slashState;
- (NSString *)slashFilterText;
- (NSDictionary *)codeBlockAt:(NSUInteger)location;
- (NSRange)structuralBlockRangeAt:(NSUInteger)location;
- (BOOL)moveBlock:(BOOL)down;
- (NSArray *)structuralBlocks;
- (NSRect)blockRectForRange:(NSRange)range;
- (void)updateFormatBar;
- (void)formatBarHide;
- (void)formatBarParentChanged:(NSNotification *)note;
- (void)formatBarApply:(NSString *)command;
- (void)formatBarLink:(NSString *)url;
- (NSDictionary *)formatBarState;
- (BOOL)formatBarSnapshotTo:(NSString *)path;
- (void)tableCommand:(NSString *)command;
- (BOOL)tableMoveRow:(NSInteger)row toIndex:(NSInteger)target inTable:(NSTextTable *)table;
- (BOOL)tableMoveColumn:(NSInteger)column toIndex:(NSInteger)target inTable:(NSTextTable *)table;
- (void)replaceTable:(NSRange)range paragraph:(NSDictionary *)p row:(NSInteger)row column:(NSInteger)column;
- (BOOL)pasteTableText:(NSString *)text;
- (BOOL)replaceAcrossTable:(NSRange)range withText:(NSString *)text;
- (void)indentList:(BOOL)outdent;
- (BOOL)indentCode:(BOOL)outdent;
- (void)codeNewline;
- (BOOL)leaveCodeBlock;
- (NSDictionary *)listContinuationAfterCode:(NSDictionary *)code before:(NSUInteger)location;
- (BOOL)replaceCodeRange:(NSRange)range withText:(NSString *)text;
- (BOOL)replaceCodeRange:(NSRange)range withText:(NSString *)text newlineCommand:(BOOL)newlineCommand;
- (void)setCodeLanguage:(NSString *)language;
- (void)setCodeLanguage:(NSString *)language atLocation:(NSUInteger)location;
- (CGFloat)codeGutterWidth;
- (void)drawCodeLineNumbers:(NSRect)dirty range:(NSRange)within;
- (NSDictionary *)codeControlState;
- (NSArray *)codeBlocks;
- (BOOL)indentQuote:(BOOL)outdent;
- (void)continueListParagraph;
- (void)beginStructuralCommand;
- (NSArray *)mathForDocument:(NSDictionary *)document;
- (NSArray *)mathForFragment:(NSDictionary *)document replacingRange:(NSRange)range;
- (NSDictionary *)nextMath;
- (NSImage *)mathImage:(NSDictionary *)result;
- (void)refreshMath;
- (void)finishInlineSource:(BOOL)force;
- (BOOL)revealInlineMathAt:(NSUInteger)location;
- (BOOL)revealInlineAt:(NSUInteger)location;
- (BOOL)prepareInlineReplacement:(NSRange *)range;
- (BOOL)finishSourceEditing;
- (void)refreshAllMath;
- (void)replace:(NSRange)range with:(NSAttributedString *)value selection:(NSRange)selection;
- (void)restore:(NSAttributedString *)value selection:(NSRange)selection typing:(NSDictionary *)typing ending:(NSDictionary *)ending;
- (NSDictionary *)document;
- (NSDictionary *)documentForRange:(NSRange)selection;
- (NSAttributedString *)attributedDocument:(NSDictionary *)document;
- (NSAttributedString *)attributedDocument:(NSDictionary *)document replacingRange:(NSRange)range;
- (void)newline; - (BOOL)leaveQuotedList:(NSRange)paragraph;
- (void)applyInputRules;
- (void)renumber;
- (void)updateTypingParagraph;
- (BOOL)moveTableCell:(BOOL)backward;
- (NSAttributedString *)attributedRuns:(NSArray *)runs paragraph:(NSDictionary *)paragraph extra:(NSDictionary *)extra;
- (void)toggleCheckbox:(NSUInteger)location;
- (NSRect)markerRect:(NSRange)range;
- (NSRect)markerLineRect:(NSRange)range;
- (NSAttributedString *)figure:(NSDictionary *)p;
- (NSAttributedString *)inlineMath:(NSDictionary *)run paragraph:(NSDictionary *)paragraph extra:(NSDictionary *)extra;
- (NSAttributedString *)sourceText:(NSDictionary *)p;
- (void)revealSourceAt:(NSUInteger)location;
- (NSRange)draftGroupAt:(NSUInteger)location;
- (void)renderDraftGroup:(NSRange)group;
- (void)finishDraftGroup;
- (NSRect)imageBounds:(NSImage *)img spec:(NSString *)spec;
- (void)refreshAlerts;
- (void)setQuoteAlert:(NSString *)label;
- (NSArray *)imageFigureRanges;
- (NSDictionary *)imageFigureAt:(NSUInteger)caret;
- (NSString *)imageSpecAt:(NSUInteger)caret;
- (void)setImageSize:(NSString *)spec;
- (void)setImageSize:(NSString *)spec atLocation:(NSUInteger)location;
- (BOOL)replaceImageAtLocation:(NSUInteger)location withURL:(NSString *)url;
- (NSArray *)tocFigureRanges;
- (NSAttributedString *)pageFigure:(NSDictionary *)p;
- (NSDictionary *)pageCardState;
- (BOOL)openDocumentLink:(id)link;
- (BOOL)deleteMentionBefore:(NSUInteger)location;
- (NSArray *)tocEntriesFromText;
- (void)refreshTOC;
- (NSInteger)tocEntryAtPoint:(NSPoint)point range:(NSRange)range;
- (void)jumpToTOCEntry:(NSInteger)entry;
@end
@interface CREditor (Parity)
- (void)setFocusMode:(BOOL)enabled;
- (void)setTypewriter:(BOOL)enabled;
- (void)showFindBar;
- (void)showReplaceBar;
- (void)revealHeading:(NSInteger)index;
- (void)printDocument;
- (BOOL)emojiRule:(BOOL)boundary;
- (BOOL)autoPair:(id)value range:(NSRange)range;
- (void)insertMarkdownText:(NSString *)markdown;
- (void)applyFocusMode;
- (void)centerCaretLine;
@end

// CRParagraphDict is paragraph metadata as stored in text attributes. It
// behaves as a plain dictionary but hashes on identifying content: the
// stock NSDictionary hash is just its count, so thousands of paragraph
// dictionaries collide in AppKit's attribute uniquing table and loading a
// long note turns quadratic.
@interface CRParagraphDict : NSDictionary
- (instancetype)initWithParagraph:(NSDictionary *)paragraph;
@end
@implementation CRParagraphDict { NSDictionary *_d; NSUInteger _hash; }
- (instancetype)initWithParagraph:(NSDictionary *)paragraph {
    if ((self=[super init])) {
        _d=[paragraph copy];
        NSUInteger h=_d.count;
        for (NSString *key in @[@"kind",@"id",@"raw",@"listItem",@"footnoteGroup",@"level",@"depth",@"ordinal",@"open",@"quote"]) h=h*31+[_d[key] hash];
        _hash=h;
    }
    return self;
}
- (NSUInteger)count { return _d.count; }
- (id)objectForKey:(id)key { return [_d objectForKey:key]; }
- (NSEnumerator *)keyEnumerator { return [_d keyEnumerator]; }
- (NSUInteger)hash { return _hash; }
- (id)copyWithZone:(NSZone *)zone { return self; }
@end
static NSDictionary *CRStoredParagraph(NSDictionary *p) {
    return !p || [p isKindOfClass:CRParagraphDict.class] ? p : [[CRParagraphDict alloc] initWithParagraph:p];
}
static NSDictionary *CRJSON(const char *json);
static BOOL CRRuleText(NSString *text);
static NSDictionary *CRTextAttrs(NSDictionary *attrs);
static BOOL CRFigure(NSDictionary *p);
static NSDictionary *CRBody(void) { return @{@"kind": @"body", @"runs": @[]}; }
// CRStaleMarker: a revealed rule or [TOC] line edited into other text. It is
// prose now (Typora), not raw source.
static BOOL CRStaleMarker(NSDictionary *p, NSString *text) {
    NSString *raw=[p[@"raw"] isKindOfClass:NSString.class] ? p[@"raw"] : nil;
    if (![p[@"kind"] isEqual:@"source"] || !raw) return NO;
    if (CRRuleText(raw)) return !CRRuleText(text);
    BOOL toc=[[raw stringByTrimmingCharactersInSet:NSCharacterSet.whitespaceCharacterSet] caseInsensitiveCompare:@"[toc]"]==NSOrderedSame;
    return toc && [[text stringByTrimmingCharactersInSet:NSCharacterSet.whitespaceCharacterSet] caseInsensitiveCompare:@"[toc]"]!=NSOrderedSame;
}
// CRBlockEnding: the paragraph's source separator holds a blank line, so it
// is the last line of its block.
static BOOL CRBlockEnding(NSDictionary *p) {
    NSString *separator = [p[@"separator"] isKindOfClass:NSString.class] ? p[@"separator"] : @"";
    return [[separator stringByReplacingOccurrencesOfString:@"\r" withString:@""] rangeOfString:@"\n\n"].location != NSNotFound;
}
static BOOL CRKind(NSDictionary *p, NSString *kind) { return [p[@"kind"] isEqual:kind]; }
static BOOL CRList(NSDictionary *p) { return CRKind(p,@"bullet") || CRKind(p,@"ordered") || CRKind(p,@"checklist"); }
static NSInteger CRQuoteOf(NSDictionary *p) { return MAX(0,[p[@"quote"] integerValue]); }
// CRPairCloser is the closing half auto-inserted after an opening bracket
// or quote, ASCII and full-width CJK alike.
static NSString *CRPairCloser(NSString *open) {
    static NSDictionary *pairs;
    if (!pairs) pairs=@{@"(":@")",@"[":@"]",@"{":@"}",@"\"":@"\"",@"'":@"'",@"（":@"）",@"【":@"】",@"《":@"》",@"“":@"”",@"‘":@"’",@"「":@"」",@"『":@"』"};
    return pairs[open];
}
// CRCodeListIndent is the text column of the list item a code line belongs
// to (0 outside lists), in points before zoom.
static CGFloat CRCodeListIndent(NSDictionary *p) {
    if (![p[@"codeIndent"] length] && ![p[@"listItem"] length]) return 0;
    if ([p[@"kind"] isEqual:@"quote"]) return [p[@"codeIndent"] length] ? 30+MAX(0,[p[@"itemDepth"] integerValue])*26 : 0;
    if (![p[@"kind"] isEqual:@"code"]) return 0;
    return 30+(MAX(0,[p[@"depth"] integerValue])+CRQuoteOf(p))*26;
}
static BOOL CROrdered(NSDictionary *p) { return CRKind(p,@"ordered") || (CRKind(p,@"checklist") && [p[@"ordinal"] integerValue]>0); }
static NSDictionary *CRFresh(NSDictionary *p) {
    NSMutableDictionary *fresh = [@{@"kind": p[@"kind"] ?: @"body", @"runs": @[]} mutableCopy];
    if (p[@"level"]) fresh[@"level"] = p[@"level"];
    if (p[@"depth"]) fresh[@"depth"] = p[@"depth"];
    if ([p[@"alert"] length]) fresh[@"alert"] = p[@"alert"];
    if (p[@"listId"]) fresh[@"listId"] = p[@"listId"];
    if (p[@"listContainer"]) fresh[@"listContainer"] = p[@"listContainer"];
    if (p[@"listMarker"]) fresh[@"listMarker"] = p[@"listMarker"];
    if (CRList(p) && p[@"quote"]) fresh[@"quote"] = p[@"quote"];
    if (CRKind(p,@"quote") && [p[@"codeIndent"] length]) for (NSString *key in @[@"codeIndent",@"itemDepth",@"listItem",@"listContinuation",@"listMarker"]) if (p[key]) fresh[key]=p[key];
    if (CRKind(p,@"code")) for (NSString *key in @[@"codeNewline",@"codeClose",@"codeSeparator",@"listItem",@"listContinuation",@"codeIndent",@"listMarker"]) if (p[key]) fresh[key]=p[key];
    if ([p[@"footnoteId"] length]) {
        for (NSString *key in @[@"footnoteId",@"footnoteGroup",@"footnoteMarker",@"footnoteIndent"]) if (p[key]) fresh[key]=p[key];
        fresh[@"footnoteContinuation"]=@YES;
        if (p[@"separator"]) fresh[@"separator"]=p[@"separator"];
    }
    if (CRList(p)) fresh[@"listItem"]=NSUUID.UUID.UUIDString;
    if (CROrdered(p)) fresh[@"ordinal"] = @([p[@"ordinal"] integerValue] + 1);
    return fresh;
}
static NSDictionary *CRInFootnote(NSDictionary *paragraph, NSDictionary *container) {
    if (![container[@"footnoteId"] length]) return paragraph;
    NSMutableDictionary *p=[paragraph mutableCopy];
    for (NSString *key in @[@"footnoteId",@"footnoteGroup",@"footnoteMarker",@"footnoteIndent"]) if (container[key]) p[key]=container[key];
    [p removeObjectForKey:@"footnoteRaw"]; [p removeObjectForKey:@"footnoteBaseline"];
    p[@"footnoteContinuation"]=@YES;
    if (![p[@"separator"] length] && container[@"separator"]) p[@"separator"]=container[@"separator"];
    return p;
}
static NSArray<NSValue *> *CRRanges(NSString *string) {
    NSMutableArray *result = [NSMutableArray array];
    NSUInteger position = 0;
    while (position < string.length) {
        NSRange r = [string paragraphRangeForRange:NSMakeRange(position,0)];
        [result addObject:[NSValue valueWithRange:r]];
        position = NSMaxRange(r);
    }
    if (string.length == 0 || [string hasSuffix:@"\n"]) [result addObject:[NSValue valueWithRange:NSMakeRange(string.length,0)]];
    return result;
}
// CRRangesIn lists the paragraph ranges that intersect range (plus the
// trailing empty paragraph when range reaches the end), so drawing can walk
// just the visible part of a long note.
static NSArray<NSValue *> *CRRangesIn(NSString *string, NSRange range) {
    NSMutableArray *result=[NSMutableArray array];
    NSUInteger end=MIN(NSMaxRange(range),string.length);
    NSUInteger position=[string paragraphRangeForRange:NSMakeRange(MIN(range.location,string.length),0)].location;
    while (position<string.length && position<=end) {
        NSRange r=[string paragraphRangeForRange:NSMakeRange(position,0)];
        [result addObject:[NSValue valueWithRange:r]];
        position=NSMaxRange(r);
    }
    if (end>=string.length && (string.length==0 || [string hasSuffix:@"\n"])) [result addObject:[NSValue valueWithRange:NSMakeRange(string.length,0)]];
    return result;
}
static NSRange CRContent(NSString *string, NSRange range) {
    while (range.length && [[NSCharacterSet newlineCharacterSet] characterIsMember:[string characterAtIndex:NSMaxRange(range)-1]]) range.length--;
    return range;
}
static NSDictionary *CRParagraphAt(CREditor *e, NSUInteger location) {
    if (location < e.text.textStorage.length) return [e.text.textStorage attribute:CRParagraph atIndex:location effectiveRange:NULL] ?: CRBody();
    return e.emptyParagraph ?: e.text.typingAttributes[CRParagraph] ?: CRBody();
}
static CGFloat CRTopInset(CREditor *editor) {
    return (22+(CRKind(CRParagraphAt(editor,0),@"code") ? 20 : 0))*editor.zoom;
}
// CRMutedColor is secondary ink for code metadata and raw markup. It mirrors
// the shell's TextMuted tokens so both editing surfaces share one hierarchy.
static NSColor *CRMutedColor(BOOL dark) {
    return dark ? [NSColor colorWithRed:0.780 green:0.800 blue:0.831 alpha:1]
                : [NSColor colorWithWhite:0.40 alpha:1];
}

// CRTintedSymbol returns an SF Symbol drawn in tint. Template drawing alone
// ignores the fill color on modern macOS, so the palette configuration does
// the tinting; macOS 11 falls back to a template image.
static NSImage *CRTintedSymbol(NSString *name, CGFloat size, NSFontWeight weight, NSColor *tint) {
    NSImage *symbol = [NSImage imageWithSystemSymbolName:name accessibilityDescription:nil];
    if (!symbol) return nil;
    NSImageSymbolConfiguration *config = [NSImageSymbolConfiguration configurationWithPointSize:size weight:weight];
    if (@available(macOS 12.0, *)) {
        config = [config configurationByApplyingConfiguration:[NSImageSymbolConfiguration configurationWithHierarchicalColor:tint]];
        return [symbol imageWithSymbolConfiguration:config];
    }
    symbol = [symbol imageWithSymbolConfiguration:config];
    symbol.template = YES;
    return symbol;
}

// CRHlColor maps a chroma token class (see render.Highlight) to a theme color.
// A one-palette-each-mode scheme keeps the code chip legible without a full
// theme engine; unknown classes inherit the default source color (nil).
static NSColor *CRHlColor(NSString *cls, BOOL dark) {
    static NSDictionary *lightMap=nil,*darkMap=nil;
    if (!lightMap) {
        lightMap=@{
            @"keyword":[NSColor colorWithRed:0.61 green:0.14 blue:0.58 alpha:1],
            @"string":[NSColor colorWithRed:0.11 green:0.46 blue:0.16 alpha:1],
            @"number":[NSColor colorWithRed:0.11 green:0.30 blue:0.75 alpha:1],
            @"literal":[NSColor colorWithRed:0.11 green:0.30 blue:0.75 alpha:1],
            @"comment":CRMutedColor(NO),
            @"operator":[NSColor colorWithRed:0.40 green:0.22 blue:0.56 alpha:1],
            @"type":[NSColor colorWithRed:0.19 green:0.42 blue:0.66 alpha:1],
            @"name":[NSColor colorWithRed:0.47 green:0.30 blue:0.07 alpha:1],
        };
        darkMap=@{
            @"keyword":[NSColor colorWithRed:0.86 green:0.56 blue:0.90 alpha:1],
            @"string":[NSColor colorWithRed:0.58 green:0.82 blue:0.52 alpha:1],
            @"number":[NSColor colorWithRed:0.60 green:0.76 blue:0.98 alpha:1],
            @"literal":[NSColor colorWithRed:0.60 green:0.76 blue:0.98 alpha:1],
            @"comment":CRMutedColor(YES),
            @"operator":[NSColor colorWithRed:0.80 green:0.68 blue:0.92 alpha:1],
            @"type":[NSColor colorWithRed:0.56 green:0.78 blue:0.96 alpha:1],
            @"name":[NSColor colorWithRed:0.90 green:0.74 blue:0.46 alpha:1],
        };
    }
    return (dark ? darkMap : lightMap)[cls];
}
static NSTextTableBlock *CRCellAt(CREditor *editor, NSUInteger location) {
    if (location >= editor.text.textStorage.length) return nil;
    return [editor.text.textStorage attribute:CRTableCell atIndex:location effectiveRange:NULL];
}
// CRTableRangeNear expands from a character inside the table cell by cell,
// so its cost is the table's size, not the document's.
static NSRange CRTableRangeNear(CREditor *editor, NSTextTable *table, NSUInteger hint) {
    NSTextStorage *storage=editor.text.textStorage;
    NSUInteger length=storage.length;
    if (hint>=length || [[storage attribute:CRTableCell atIndex:hint effectiveRange:NULL] table]!=table) return NSMakeRange(NSNotFound,0);
    NSRange cell;
    [storage attribute:CRTableCell atIndex:hint longestEffectiveRange:&cell inRange:NSMakeRange(0,length)];
    NSUInteger start=cell.location, end=NSMaxRange(cell);
    while (start>0) {
        NSRange previous;
        NSTextTableBlock *block=[storage attribute:CRTableCell atIndex:start-1 longestEffectiveRange:&previous inRange:NSMakeRange(0,start)];
        if (block.table!=table) break;
        start=previous.location;
    }
    while (end<length) {
        NSRange next;
        NSTextTableBlock *block=[storage attribute:CRTableCell atIndex:end longestEffectiveRange:&next inRange:NSMakeRange(end,length-end)];
        if (block.table!=table) break;
        end=NSMaxRange(next);
    }
    return NSMakeRange(start,end-start);
}
// CRTableSpan covers every table touching range (or adjacent to it) in text.
static NSRange CRTableSpan(NSAttributedString *text, NSRange range) {
    NSUInteger length=text.length;
    NSUInteger from=range.location>0 ? range.location-1 : 0, to=MIN(length,NSMaxRange(range)+1);
    __block NSRange result=NSMakeRange(NSNotFound,0);
    if (to<=from) return result;
    [text enumerateAttribute:CRTableCell inRange:NSMakeRange(from,to-from) options:0 usingBlock:^(NSTextTableBlock *cell, NSRange part, BOOL *stop) {
        if (!cell) return;
        NSUInteger start=part.location, end=NSMaxRange(part);
        while (start>0) {
            NSRange previous;
            NSTextTableBlock *block=[text attribute:CRTableCell atIndex:start-1 longestEffectiveRange:&previous inRange:NSMakeRange(0,start)];
            if (block.table!=cell.table) break;
            start=previous.location;
        }
        while (end<length) {
            NSRange next;
            NSTextTableBlock *block=[text attribute:CRTableCell atIndex:end longestEffectiveRange:&next inRange:NSMakeRange(end,length-end)];
            if (block.table!=cell.table) break;
            end=NSMaxRange(next);
        }
        NSRange table=NSMakeRange(start,end-start);
        result=result.location==NSNotFound ? table : NSUnionRange(result,table);
    }];
    return result;
}
static NSRange CRTableRange(CREditor *editor, NSTextTable *table) {
    __block NSRange result = NSMakeRange(NSNotFound, 0);
    [editor.text.textStorage enumerateAttribute:CRTableCell inRange:NSMakeRange(0,editor.text.string.length) options:0 usingBlock:^(NSTextTableBlock *cell, NSRange range, BOOL *stop) {
        if (cell.table == table) result = result.location == NSNotFound ? range : NSUnionRange(result, range);
    }];
    return result;
}
static NSArray *CRSelectedParagraphs(CREditor *e) {
    NSRange selection = e.text.selectedRange;
    NSMutableArray *ranges = [NSMutableArray array];
    for (NSValue *v in CRRanges(e.text.string)) {
        NSRange r = v.rangeValue;
        if (selection.length ? NSIntersectionRange(r,selection).length > 0 : (selection.location >= r.location && (r.length == 0 || selection.location < NSMaxRange(r) || (NSMaxRange(r) == e.text.string.length && ![e.text.string hasSuffix:@"\n"])))) {
            [ranges addObject:v];
            if (!selection.length) break;
        }
    }
    return ranges;
}

static NSArray *CRImportedParagraphs(NSArray *paragraphs) {
    NSMutableArray *result=[NSMutableArray array];
    NSMutableDictionary *identifiers=[NSMutableDictionary dictionary];
    NSMutableSet *items=[NSMutableSet set];
    NSInteger base=-1;
    for (NSDictionary *original in paragraphs) {
        NSMutableDictionary *p=[original mutableCopy];
        [p removeObjectForKey:@"id"]; [p removeObjectForKey:@"nextId"];
        if ([p[@"footnoteGroup"] length]) {
            NSString *key=[@"footnote:" stringByAppendingString:p[@"footnoteGroup"]];
            if (!identifiers[key]) identifiers[key]=NSUUID.UUID.UUIDString;
            p[@"footnoteGroup"]=identifiers[key];
        }
        if (CRList(p)) {
            NSString *owner=p[@"listItem"] ?: @"";
            if ([p[@"listContinuation"] boolValue] && ![items containsObject:owner]) [p removeObjectForKey:@"listContinuation"];
            [items addObject:owner];
            NSInteger depth=MAX(0,[p[@"depth"] integerValue]);
            base=base<0 ? depth : MIN(base,depth);
            p[@"depth"]=@(depth-base);
            // Clipboard fragments form their own forest, without the omitted
            // ancestors or identities belonging to the source document.
            for (NSString *field in @[@"listId",@"listContainer",@"listItem"]) {
                NSString *key=[NSString stringWithFormat:@"%@:%@",field,p[field] ?: @""];
                if (!identifiers[key]) identifiers[key]=NSUUID.UUID.UUIDString;
                p[field]=identifiers[key];
            }
            [p removeObjectForKey:@"listBaseline"]; [p removeObjectForKey:@"listIndent"];
        } else base=-1;
        [result addObject:p];
    }
    return result;
}
// CRLocalDocument reports whether a link target is a note next to this one
// ("other.md", "dir/note.markdown#part"), not a web page or an anchor.
static BOOL CRLocalDocument(id target) {
    NSString *link=[target isKindOfClass:NSURL.class] ? [(NSURL *)target absoluteString] : [target isKindOfClass:NSString.class] ? target : nil;
    if (!link.length || [link hasPrefix:@"#"] || [link rangeOfString:@":"].location!=NSNotFound) return NO;
    NSString *path=[[link componentsSeparatedByString:@"#"].firstObject stringByRemovingPercentEncoding] ?: link;
    NSString *ext=path.pathExtension.lowercaseString;
    return [ext isEqual:@"md"] || [ext isEqual:@"markdown"];
}
// CRIsMention: a link to a note whose text starts with "@".
static BOOL CRIsMention(NSString *text, id link) {
    return [text isKindOfClass:NSString.class] && text.length>1 && [text hasPrefix:@"@"] && CRLocalDocument(link);
}
static BOOL CRHasFootnote(NSDictionary *paragraph) {
    if ([paragraph[@"footnoteId"] length]) return YES;
    for (NSDictionary *run in paragraph[@"runs"]) if ([run[@"footnote"] length]) return YES;
    for (NSArray *row in paragraph[@"rows"]) for (NSDictionary *cell in row) for (NSDictionary *run in cell[@"runs"]) if ([run[@"footnote"] length]) return YES;
    return NO;
}

// The slash menu and format bar panels live at file scope; their editor halves
// are included inside @implementation CREditor below.
#include "color_palette_darwin.h"
#include "slash_menu_view_darwin.h"
#include "format_bar_view_darwin.h"
#include "footnote_card_view_darwin.h"
#include "emoji_menu_view_darwin.h"
#include "link_menu_view_darwin.h"

@implementation CREditor
- (void)notify { if (!_suppress && _callback) { if (!_text.hasMarkedText) { [self refreshMath]; [self refreshTOC]; } cr_callback(_callback,0); } }
- (void)beginStructuralCommand {
    [_text breakUndoCoalescing];
    // AppKit owns automatic event groups, including the IME's pending edits.
}
- (void)source { if (!_suppress && _callback) cr_callback(_callback,1); }
- (NSUndoManager *)undoManagerForTextView:(NSTextView *)view { return _undo; }
// In-page "#heading" links scroll to their heading, as in Typora; other
// links keep NSTextView's default of opening in the system.
- (BOOL)textView:(NSTextView *)view clickedOnLink:(id)link atIndex:(NSUInteger)index {
    // Typora: a click edits the link text, Cmd-click follows it.
    NSEvent *event=NSApp.currentEvent;
    BOOL follow=!view.editable || !event || event.type!=NSEventTypeLeftMouseUp && event.type!=NSEventTypeLeftMouseDown || (event.modifierFlags & NSEventModifierFlagCommand);
    if (!follow) {
        view.selectedRange=NSMakeRange(index,0);
        [self revealInlineAt:index];
        return YES;
    }
    NSString *target=[link isKindOfClass:NSURL.class] ? [(NSURL *)link absoluteString] : [link description];
    if ([self openDocumentLink:link]) return YES; // a note opens in Corra, not in another app
    if (![target hasPrefix:@"#"]) return NO;
    [self jumpToAnchor:[target substringFromIndex:1]];
    return YES; // never hand a bare fragment to the system
}
- (NSDictionary *)visual:(NSDictionary *)attrs paragraph:(NSDictionary *)p {
    NSMutableDictionary *out = [attrs mutableCopy];
    if (out[CRParagraph]) out[CRParagraph]=CRStoredParagraph(out[CRParagraph]);
    // Notes-like scale: a quiet 16pt body; weight, not size jumps, carries
    // the hierarchy. Sizes pair with the spacing rhythm below: bigger blocks
    // get more air before them than after, so a heading belongs to the
    // section it opens.
    static const CGFloat headingSizes[] = {30, 24, 19.5, 17, 16, 15};
    static const CGFloat headingBefore[] = {10, 26, 20, 16, 14, 12};
    static const CGFloat headingAfter[] = {12, 8, 6, 4, 4, 4};
    BOOL heading = CRKind(p,@"heading");
    NSInteger level = [p[@"level"] integerValue];
    if (level < 1) level = 1;
    if (level > 6) level = 6;
    BOOL source = CRKind(p,@"source") || CRKind(p,@"code"); // editable code/source lines share the mono style
    NSTextTableBlock *cell = attrs[CRTableCell];
    CGFloat size = 16 * _zoom;
    if (heading) size = headingSizes[level-1] * _zoom;
    if (cell || source) size = 13.5 * _zoom;
    // Footnote definitions read as notes, not body text.
    BOOL note = [p[@"footnoteId"] length] > 0 && !heading && !source;
    if (note) size = 14 * _zoom;
    if (([attrs[CRCode] boolValue] || [attrs[CRKbd] boolValue]) && !heading) size -= 1.5 * _zoom;
    BOOL mono = [attrs[CRCode] boolValue] || [attrs[CRKbd] boolValue] || source;
    NSFont *font;
    if (mono) {
        font = _monoFont.length ? [NSFont fontWithName:_monoFont size:size] : nil;
        if (!font) font = [NSFont monospacedSystemFontOfSize:size weight:NSFontWeightRegular];
    } else {
        font = _bodyFont.length ? [NSFont fontWithName:_bodyFont size:size] : nil;
        if (!font) font = [NSFont systemFontOfSize:size weight:(heading && level > 2) ? NSFontWeightSemibold : NSFontWeightRegular];
    }
    NSFontTraitMask traits = 0;
    if ([attrs[CRBold] boolValue] || (heading && level <= 2) || (cell && cell.startingRow == 0)) traits |= NSBoldFontMask;
    if (traits) font = [[NSFontManager sharedFontManager] convertFont:font toHaveTrait:traits];
    // Italic is a synthetic slant, as in Typora: CJK fonts ship no italic
    // face, and SF's italic is itself an oblique, so both scripts match.
    if ([attrs[CRItalic] boolValue]) out[NSObliquenessAttributeName]=@0.2;
    else [out removeObjectForKey:NSObliquenessAttributeName];
    // Sup/sub shift the baseline and shrink the glyph; scrub the display
    // attribute first so a removed semantic trait does not linger.
    BOOL sup = [attrs[CRSup] boolValue], sub = [attrs[CRSub] boolValue];
    if (sup || sub) {
        font = [NSFont fontWithDescriptor:font.fontDescriptor size:font.pointSize * 0.75];
        out[NSSuperscriptAttributeName] = sup ? @1 : @-1;
    } else [out removeObjectForKey:NSSuperscriptAttributeName];
    out[NSFontAttributeName] = font;
    // Hovering a link shows where it goes and how to get there.
    id link=attrs[NSLinkAttributeName];
    if (link && !attrs[CRMathRendered]) out[NSToolTipAttributeName]=[NSString stringWithFormat:@"%@\n⌘ 点按以打开",[link isKindOfClass:NSURL.class] ? [link absoluteString] : link];
    else if (!attrs[CRMathRendered] && !attrs[CRFootnote]) [out removeObjectForKey:NSToolTipAttributeName];
    out[NSForegroundColorAttributeName] = _dark ? [NSColor colorWithRed:0.906 green:0.914 blue:0.933 alpha:1] : [NSColor colorWithWhite:0.161 alpha:1];
    // Quotes step back: same size, quieter ink, so the bar and the tint carry
    // the "cited, not mine" reading instead of a smaller font.
    if ([attrs[CRHTML] boolValue]) {
        // Raw HTML reads as markup, not prose: quiet and monospaced.
        out[NSFontAttributeName] = [NSFont monospacedSystemFontOfSize:size-2*_zoom weight:NSFontWeightRegular];
        out[NSForegroundColorAttributeName] = CRMutedColor(_dark);
    } else if (CRKind(p,@"quote") || note || (CRList(p) && CRQuoteOf(p))) out[NSForegroundColorAttributeName] = _dark ? [NSColor colorWithWhite:0.62 alpha:1] : [NSColor colorWithWhite:0.42 alpha:1];
    NSMutableParagraphStyle *style = [NSMutableParagraphStyle new];
    // ~1.55 line height keeps mixed CJK/Latin lines from rubbing shoulders.
    style.lineSpacing = 6 * _zoom;
    if (CRKind(p,@"math")) style.alignment=NSTextAlignmentCenter;
    // Lists hang together as a group; paragraphs breathe more.
    BOOL list = CRKind(p,@"bullet") || CRKind(p,@"ordered") || CRKind(p,@"checklist");
    style.paragraphSpacing = (list ? 6 : 12) * _zoom;
    if (heading) {
        style.paragraphSpacingBefore = headingBefore[level-1] * _zoom;
        style.paragraphSpacing = headingAfter[level-1] * _zoom;
    }
    if (source) {
        style.paragraphSpacing = (CRBlockEnding(p) ? 16 : 3) * _zoom;
        style.lineSpacing = 3 * _zoom;
    }
    if (CRKind(p,@"code") && [p[@"open"] length]) style.paragraphSpacingBefore=12*_zoom;
    // The chip reaches 12pt past the text; keep a clear gap below it.
    if (CRKind(p,@"code") && [p[@"close"] length]) style.paragraphSpacing=26*_zoom;
    // A callout card overhangs its text by 6pt each way; the last line of
    // one leaves room so the next block (or card) does not touch it.
    if ((CRKind(p,@"quote") || (list && CRQuoteOf(p))) && [p[@"alert"] length] && CRBlockEnding(p)) style.paragraphSpacing=22*_zoom;
    else if (list && CRQuoteOf(p) && CRBlockEnding(p)) style.paragraphSpacing=MAX(style.paragraphSpacing,12*_zoom);
    if (cell) {
        style.textBlocks = @[cell];
        style.paragraphSpacing = 0;
        style.lineSpacing = 2 * _zoom;
        NSArray *alignments = p[@"alignments"];
        NSString *alignment = cell.startingColumn < alignments.count ? alignments[cell.startingColumn] : @"none";
        style.alignment = [alignment isEqual:@"center"] ? NSTextAlignmentCenter : [alignment isEqual:@"right"] ? NSTextAlignmentRight : NSTextAlignmentLeft;
        // Airy horizontal padding, compact rows, hairline grid; the header
        // gets a fill and a stronger bottom rule instead of a heavier grid.
        [cell setWidth:6*_zoom type:NSTextBlockAbsoluteValueType forLayer:NSTextBlockPadding edge:NSMinYEdge];
        [cell setWidth:6*_zoom type:NSTextBlockAbsoluteValueType forLayer:NSTextBlockPadding edge:NSMaxYEdge];
        [cell setWidth:10*_zoom type:NSTextBlockAbsoluteValueType forLayer:NSTextBlockPadding edge:NSMinXEdge];
        [cell setWidth:10*_zoom type:NSTextBlockAbsoluteValueType forLayer:NSTextBlockPadding edge:NSMaxXEdge];
        [cell setWidth:0.5 type:NSTextBlockAbsoluteValueType forLayer:NSTextBlockBorder];
        [cell setBorderColor:_dark ? [NSColor colorWithWhite:0.30 alpha:1] : [NSColor colorWithWhite:0.84 alpha:1]];
        if (cell.startingRow == 0) {
            [cell setWidth:1 type:NSTextBlockAbsoluteValueType forLayer:NSTextBlockBorder edge:NSMaxYEdge];
            [cell setBorderColor:_dark ? [NSColor colorWithWhite:0.45 alpha:1] : [NSColor colorWithWhite:0.70 alpha:1] forEdge:NSMaxYEdge];
        }
        cell.backgroundColor = cell.startingRow == 0 ? (_dark ? [NSColor colorWithWhite:0.235 alpha:1] : [NSColor colorWithWhite:0.95 alpha:1]) : NSColor.clearColor;
    }
    if (list || CRKind(p,@"quote")) {
        style.firstLineHeadIndent = (30 + (MAX(0,[p[@"depth"] integerValue])+(list ? CRQuoteOf(p) : 0))*26 + (CRKind(p,@"checklist") && CROrdered(p) ? 24 : 0) + (list ? 0 : CRCodeListIndent(p))) * _zoom;
        style.headIndent = style.firstLineHeadIndent;
    }
    if (source) {
        // Code sits inside its chip with breathing room on both sides.
        CGFloat item=CRCodeListIndent(p);
        style.firstLineHeadIndent=(14+item)*_zoom; style.headIndent=(14+item)*_zoom; style.tailIndent=-14*_zoom;
    }
    if (CRKind(p,@"code") && _codeLineNumbers) {
        style.firstLineHeadIndent=[self codeGutterWidth];
        style.headIndent=style.firstLineHeadIndent;
    }
    if ([p[@"footnoteId"] length]) {
        style.firstLineHeadIndent+=30*_zoom; style.headIndent+=30*_zoom;
    }
    out[NSParagraphStyleAttributeName] = style;
    [out removeObjectForKey:NSBackgroundColorAttributeName];
    [out removeObjectForKey:NSUnderlineStyleAttributeName];
    if ([attrs[CRUnderline] boolValue]) out[NSUnderlineStyleAttributeName] = @(NSUnderlineStyleSingle);
    // Inline code, key caps and highlights are rounded chips painted by
    // drawInlineChips: a glyph background fills the whole line height.
    // A callout's tint is a panel painted by drawAlertPanels, not a glyph
    // background, so wrapped and broken lines share one edge.
    if (CRKind(p,@"opaque")) {
        out[NSFontAttributeName] = [NSFont monospacedSystemFontOfSize:12.5*_zoom weight:NSFontWeightRegular];
        out[NSForegroundColorAttributeName] = _dark ? [NSColor colorWithWhite:0.65 alpha:1] : [NSColor colorWithWhite:0.43 alpha:1];
    }
    if (attrs[NSLinkAttributeName]) out[NSForegroundColorAttributeName] = NSColor.linkColor;
    if (attrs[NSLinkAttributeName] && !attrs[NSAttachmentAttributeName] && ![attrs[CRMention] boolValue]) out[NSUnderlineStyleAttributeName] = @(NSUnderlineStyleSingle);
    // A chosen text color wins over the link blue, as in Notion; raw HTML
    // and code keep their own ink.
    if ([attrs[CRMention] boolValue] && attrs[NSLinkAttributeName] && !mono) {
        // A mention reads as a reference to a page, not as a web link.
        out[NSForegroundColorAttributeName] = _dark ? [NSColor colorWithWhite:0.86 alpha:1] : [NSColor colorWithWhite:0.22 alpha:1];
        out[NSFontAttributeName] = [[NSFontManager sharedFontManager] convertWeight:YES ofFont:out[NSFontAttributeName]];
        out[NSToolTipAttributeName] = [NSString stringWithFormat:@"%@\n点按以打开", [attrs[NSLinkAttributeName] description]];
    }
    NSColor *ink = [attrs[CRHTML] boolValue] || mono ? nil : CRResolvedColor(attrs[CRColor], NO, _dark);
    if (ink) out[NSForegroundColorAttributeName] = ink;
    if (attrs[CRHl]) out[NSForegroundColorAttributeName] = CRHlColor(attrs[CRHl],_dark);
    return out;
}
- (void)restyle { _dirty=NSMakeRange(NSNotFound,0); [self restyleRange:NSMakeRange(0,_text.string.length)]; }
// markDirty records text whose paragraph styling must be recomputed.
- (void)markDirty:(NSRange)range {
    NSUInteger length=_text.string.length, location=MIN(range.location,length);
    range=NSMakeRange(location,MIN(range.length,length-location));
    _dirty=_dirty.location==NSNotFound ? range : NSUnionRange(_dirty,range);
}
// restyleDirty restyles what edits touched, plus the caret paragraph and one
// neighbour each side: typing must not cost a whole-document relayout.
- (void)restyleDirty {
    [self markDirty:_text.selectedRange];
    NSRange dirty=_dirty; _dirty=NSMakeRange(NSNotFound,0);
    NSString *string=_text.string;
    NSRange range=[string paragraphRangeForRange:dirty];
    if (range.location) range=NSUnionRange(range,[string paragraphRangeForRange:NSMakeRange(range.location-1,0)]);
    if (NSMaxRange(range)<string.length) range=NSUnionRange(range,[string paragraphRangeForRange:NSMakeRange(NSMaxRange(range),0)]);
    [self restyleRange:range];
}
// CRSameStyle compares a computed style with stored attributes. AppKit's
// font fallback swaps CJK runs to another family after the fact, so fonts
// match on size and the traits we choose, not on identity.
static BOOL CRSameStyle(NSDictionary *styled, NSDictionary *stored) {
    if (styled.count!=stored.count) return NO;
    for (NSString *key in styled) {
        id a=styled[key], b=stored[key];
        if ([key isEqual:NSFontAttributeName] && [a isKindOfClass:NSFont.class] && [b isKindOfClass:NSFont.class]) {
            NSFont *x=a, *y=b;
            NSFontDescriptorSymbolicTraits mask=NSFontDescriptorTraitBold|NSFontDescriptorTraitItalic;
            if (x.pointSize!=y.pointSize || (x.fontDescriptor.symbolicTraits&mask)!=(y.fontDescriptor.symbolicTraits&mask)) return NO;
            continue;
        }
        if (![a isEqual:b]) return NO;
    }
    return YES;
}
// CRQuoteLevel is how many quote bars a paragraph sits behind: a quote's
// depth+1, or the quotes enclosing a list item ("> - a").
static NSInteger CRQuoteLevel(NSDictionary *p) {
    if (CRKind(p,@"quote")) return MAX(0,[p[@"depth"] integerValue])+1;
    return CRList(p) ? CRQuoteOf(p) : 0;
}
// CRFootnoteAreaStart: a footnote definition whose previous paragraph is not
// one, i.e. where the footnote area begins.
static BOOL CRFootnoteAreaStart(CREditor *e, NSUInteger at) {
    if (![CRParagraphAt(e,at)[@"footnoteId"] length] || [CRParagraphAt(e,at)[@"footnoteContinuation"] boolValue]) return NO;
    return at==0 || ![CRParagraphAt(e,at-1)[@"footnoteId"] length];
}
// Code and key chips are padded beyond their glyphs; kerning the character
// before a chip and the chip's last character keeps neighbours clear of it.
- (void)spaceInlineChipsIn:(NSRange)dirty {
    NSTextStorage *storage=_text.textStorage;
    NSString *string=_text.string;
    NSMutableIndexSet *wanted=[NSMutableIndexSet indexSet];
    for (NSString *key in @[CRCode,CRKbd])
    [storage enumerateAttribute:key inRange:dirty options:0 usingBlock:^(id value, NSRange range, BOOL *stop) {
        if (![value boolValue]) return;
        NSDictionary *p=[storage attribute:CRParagraph atIndex:range.location effectiveRange:NULL];
        if (CRKind(p,@"code") || CRKind(p,@"source") || CRKind(p,@"opaque")) return;
        if (range.length) [wanted addIndex:NSMaxRange(range)-1];
        if (range.location) {
            unichar c=[string characterAtIndex:range.location-1];
            if (c!='\n' && c!=0x2028) [wanted addIndex:range.location-1];
        }
    }];
    NSNumber *kern=@(5*_zoom);
    [storage enumerateAttribute:NSKernAttributeName inRange:dirty options:0 usingBlock:^(id value, NSRange range, BOOL *stop) {
        if (!value) return;
        for (NSUInteger i=range.location;i<NSMaxRange(range);i++) if (![wanted containsIndex:i]) [storage removeAttribute:NSKernAttributeName range:NSMakeRange(i,1)];
    }];
    [wanted enumerateIndexesUsingBlock:^(NSUInteger i, BOOL *stop) {
        if (i<storage.length && ![[storage attribute:NSKernAttributeName atIndex:i effectiveRange:NULL] isEqual:kern]) [storage addAttribute:NSKernAttributeName value:kern range:NSMakeRange(i,1)];
    }];
}
- (void)restyleRange:(NSRange)dirty {
    BOOL wasSuppress = _suppress; _suppress = YES;
    NSArray *selection = _text.selectedRanges;
    BOOL whole = dirty.location==0 && dirty.length==_text.string.length;
    [self highlightCodeInRange:dirty];
    _text.textContainerInset=NSMakeSize(_text.textContainerInset.width,CRTopInset(self));
    [_text.textStorage beginEditing];
    NSAttributedString *copy = [_text.textStorage attributedSubstringFromRange:dirty];
    [copy enumerateAttributesInRange:NSMakeRange(0,copy.length) options:0 usingBlock:^(NSDictionary *attrs, NSRange range, BOOL *stop) {
        // Rewriting an unchanged run still marks it edited, and AppKit then
        // re-runs font fallback over it; only touch runs whose style moved.
        NSDictionary *styled=[self visual:attrs paragraph:attrs[CRParagraph] ?: CRBody()];
        if (!CRSameStyle(styled,attrs)) [self.text.textStorage setAttributes:styled range:NSMakeRange(dirty.location+range.location,range.length)];
    }];
    [self spaceInlineChipsIn:dirty];
    // Table rows carry no paragraph spacing, so the block after a table
    // takes the gap itself.
    BOOL afterTable=dirty.location && CRCellAt(self,dirty.location-1);
    // A callout card overhangs its first line; the block above keeps clear.
    NSString *(^alertOf)(NSUInteger)=^NSString *(NSUInteger at) {
        NSDictionary *p=CRParagraphAt(self,at);
        return CRKind(p,@"quote") && [p[@"alert"] isKindOfClass:NSString.class] ? p[@"alert"] : nil;
    };
    NSString *previousAlert=dirty.location ? alertOf(dirty.location-1) : nil;
    for (NSValue *v in CRRangesIn(_text.string,dirty)) {
        NSRange r=v.rangeValue;
        BOOL cell=r.length && CRCellAt(self,r.location);
        NSString *alert=r.length ? alertOf(r.location) : nil;
        CGFloat before=0;
        if (afterTable && !cell && r.length) before=16*_zoom;
        if (alert.length && ![alert isEqual:previousAlert] && r.location) before=MAX(before,14*_zoom);
        // The footnote area opens below a short divider (drawRect).
        if (r.length && r.location && CRFootnoteAreaStart(self,r.location)) before=MAX(before,30*_zoom);
        if (before>0) {
            NSMutableParagraphStyle *style=[[_text.textStorage attribute:NSParagraphStyleAttributeName atIndex:r.location effectiveRange:NULL] mutableCopy];
            if (style.paragraphSpacingBefore<before) {
                style.paragraphSpacingBefore=before;
                [_text.textStorage addAttribute:NSParagraphStyleAttributeName value:style range:r];
            }
        }
        afterTable=cell; previousAlert=alert;
    }
    [_text.textStorage endEditing];
    NSMutableDictionary *typing=[[self visual:_text.typingAttributes paragraph:_text.typingAttributes[CRParagraph] ?: _emptyParagraph ?: CRBody()] mutableCopy];
    [typing removeObjectForKey:NSKernAttributeName]; // chip spacing is per character
    if (typing[CRHTML]) { [typing removeObjectForKey:CRHTML]; [typing addEntriesFromDictionary:[self visual:typing paragraph:typing[CRParagraph] ?: CRBody()]]; }
    _text.typingAttributes = typing;
    _text.selectedRanges = selection;
    NSColor *background = _dark ? [NSColor colorWithRed:0.149 green:0.153 blue:0.176 alpha:1] : NSColor.whiteColor;
    _text.backgroundColor = background; _scroll.backgroundColor = background;
    _text.insertionPointColor = _dark ? NSColor.whiteColor : NSColor.blackColor;
    _scroll.appearance = [NSAppearance appearanceNamed:_dark ? NSAppearanceNameDarkAqua : NSAppearanceNameAqua];
    if (_focusMode) [self applyFocusMode];
    [self refreshRulesIn:whole ? NSMakeRange(0,_text.string.length) : dirty];
    // Numbering only changes in notes that have, or just gained, footnotes.
    if (whole || self.footnotes.count || [self hasFootnotesIn:dirty]) [self refreshFootnotes];
    [_text setNeedsDisplay:YES];
    _suppress = wasSuppress;
}
- (NSAttributedString *)attributedRuns:(NSArray *)runs paragraph:(NSDictionary *)paragraph extra:(NSDictionary *)extra {
    NSMutableAttributedString *result = [NSMutableAttributedString new];
    NSString *alert = [paragraph[@"alert"] isKindOfClass:NSString.class] ? paragraph[@"alert"] : nil;
    BOOL alertPending = alert.length > 0;
    for (NSDictionary *run in runs) {
        NSMutableDictionary *attrs = [@{CRParagraph:paragraph} mutableCopy];
        [attrs addEntriesFromDictionary:extra];
        if ([run[@"break"] boolValue]) {
            // U+2028 is a line separator: it wraps visually but keeps the whole
            // block in one -paragraphRangeForRange:, so block identity holds.
            if ([run[@"soft"] boolValue]) attrs[CRSoftBreak]=@YES;
            [result appendAttributedString:[[NSAttributedString alloc] initWithString:@" " attributes:[self visual:attrs paragraph:paragraph]]];
            continue;
        }
        if ([run[@"math"] length]) {
            NSAttributedString *m=[self inlineMath:run paragraph:paragraph extra:extra];
            if (m) { [result appendAttributedString:m]; continue; }
        }
        if ([run[@"image"] length]) {
            NSString *url=[run[@"image"] stringByRemovingPercentEncoding] ?: run[@"image"];
            NSTextAttachment *att=[self imageAttachmentForURL:url spec:nil];
            if (!att) { att=[NSTextAttachment new]; att.image=[self placeholderImage]; att.bounds=[self imageBounds:att.image spec:nil]; }
            NSMutableAttributedString *image=[[NSAttributedString attributedStringWithAttachment:att] mutableCopy];
            NSMutableDictionary *a=[[self visual:attrs paragraph:paragraph] mutableCopy];
            a[CRImageSource]=run[@"text"] ?: @""; a[CRImageURL]=run[@"image"];
            if ([run[@"link"] length]) a[NSLinkAttributeName]=run[@"link"]; // a badge opens its link
            if ([run[@"linkTitle"] length]) a[CRLinkTitle]=run[@"linkTitle"];
            [image addAttributes:a range:NSMakeRange(0,image.length)];
            [result appendAttributedString:image];
            continue;
        }
        if ([run[@"source"] boolValue]) attrs[CRInlineSource]=NSUUID.UUID.UUIDString;
        if ([run[@"bold"] boolValue]) attrs[CRBold] = @YES;
        if ([run[@"italic"] boolValue]) attrs[CRItalic] = @YES;
        if ([run[@"code"] boolValue]) attrs[CRCode] = @YES;
        if ([run[@"strike"] boolValue]) attrs[NSStrikethroughStyleAttributeName] = @(NSUnderlineStyleSingle);
        if ([run[@"underline"] boolValue]) attrs[CRUnderline] = @YES;
        if ([run[@"kbd"] boolValue]) attrs[CRKbd] = @YES;
        if ([run[@"highlight"] boolValue]) attrs[CRHighlight] = @YES;
        if ([run[@"sup"] boolValue]) attrs[CRSup] = @YES;
        if ([run[@"sub"] boolValue]) attrs[CRSub] = @YES;
        if ([run[@"color"] length]) attrs[CRColor] = run[@"color"];
        if ([run[@"background"] length]) attrs[CRBackground] = run[@"background"];
        if ([run[@"html"] boolValue]) attrs[CRHTML] = @YES;
        if ([run[@"link"] length]) attrs[NSLinkAttributeName] = run[@"link"];
        if ([run[@"linkTitle"] length]) attrs[CRLinkTitle] = run[@"linkTitle"];
        if (CRIsMention(run[@"text"],run[@"link"]) && ![run[@"code"] boolValue]) attrs[CRMention] = @YES;
        if ([run[@"footnote"] length]) {
            [result appendAttributedString:[self footnote:run attributes:[self visual:attrs paragraph:paragraph]]];
            continue;
        }
        NSString *text = run[@"text"] ?: @"";
        // A tag spanning lines ("<d\netails>") keeps its newline as a line
        // separator, or it would split the paragraph in the text view.
        if (attrs[CRHTML]) text=[text stringByReplacingOccurrencesOfString:@"\n" withString:@"\u2028"];
        if (alertPending && text.length) {
            // The marker is printed as the callout's title, not as "[!NOTE]"
            // text the reader has to decode.
            NSString *marker = [NSString stringWithFormat:@"[!%@]", alert];
            if ([text hasPrefix:marker]) {
                NSMutableDictionary *title = [attrs mutableCopy];
                title[NSFontAttributeName] = [NSFont systemFontOfSize:[attrs[NSFontAttributeName] pointSize] weight:NSFontWeightSemibold];
                title[NSForegroundColorAttributeName] = CRAlertAccent(alert,_dark);
                [result appendAttributedString:[[NSAttributedString alloc] initWithString:marker attributes:[self visual:title paragraph:paragraph]]];
                text = [text substringFromIndex:marker.length];
            }
            alertPending = NO;
            if (!text.length) continue;
        }
        NSUInteger at=result.length;
        [result appendAttributedString:[[NSAttributedString alloc] initWithString:text attributes:[self visual:attrs paragraph:paragraph]]];
        // A delimiter the source escaped (`\*`) is text: it must not pair
        // with one typed later. Inert, unescaped ones still may (Typora).
        NSString *raw=[paragraph[@"raw"] isKindOfClass:NSString.class] ? paragraph[@"raw"] : nil;
        if (![run[@"code"] boolValue] && [raw containsString:@"\\"]) {
            static NSCharacterSet *delimiters;
            if (!delimiters) delimiters=[NSCharacterSet characterSetWithCharactersInString:@"*_~=^`$[]<#\\"];
            NSRange found=[text rangeOfCharacterFromSet:delimiters];
            while (found.location!=NSNotFound) {
                NSString *escaped=[@"\\" stringByAppendingString:[text substringWithRange:found]];
                if ([raw containsString:escaped]) [result addAttribute:CRLiteral value:@YES range:NSMakeRange(at+found.location,1)];
                NSUInteger next=NSMaxRange(found);
                found=next<text.length ? [text rangeOfCharacterFromSet:delimiters options:0 range:NSMakeRange(next,text.length-next)] : NSMakeRange(NSNotFound,0);
            }
        }
    }
    return result;
}
- (NSAttributedString *)attributedDocument:(NSDictionary *)document {
    return [self attributedDocument:document replacingRange:NSMakeRange(NSNotFound,0)];
}
- (NSAttributedString *)attributedDocument:(NSDictionary *)document replacingRange:(NSRange)range {
    if (range.location==NSNotFound) [self prepareFootnotes:document];
    NSArray *previousMath=_preparedMath;
    NSUInteger previousIndex=_mathIndex;
    _preparedMath=[self mathForFragment:document replacingRange:range]; _mathIndex=0;
    NSMutableAttributedString *result = [NSMutableAttributedString new];
    NSArray *paragraphs = document[@"paragraphs"];
    if (range.location!=NSNotFound && [CRParagraphAt(self,range.location)[@"footnoteId"] length]) {
        paragraphs=[self footnoteFragment:paragraphs container:CRParagraphAt(self,range.location)];
    }
    for (NSUInteger i=0; i<paragraphs.count; i++) {
        NSDictionary *p = paragraphs[i];
        if (CRKind(p,@"table")) {
            NSArray *rows = p[@"rows"];
            NSUInteger columns = [rows.firstObject count];
            if (!columns) continue;
            NSTextTable *table = [NSTextTable new];
            table.numberOfColumns = columns; table.collapsesBorders = YES; table.hidesEmptyCells = NO;
            table.layoutAlgorithm = NSTextTableFixedLayoutAlgorithm;
            [table setValue:100 type:NSTextBlockPercentageValueType forDimension:NSTextBlockWidth];
            for (NSUInteger rowIndex=0; rowIndex<rows.count; rowIndex++) {
                NSArray *row = rows[rowIndex];
                for (NSUInteger column=0; column<columns; column++) {
                    NSTextTableBlock *cell = [[NSTextTableBlock alloc] initWithTable:table startingRow:rowIndex rowSpan:1 startingColumn:column columnSpan:1];
                    [cell setValue:100.0/columns type:NSTextBlockPercentageValueType forDimension:NSTextBlockWidth];
                    NSDictionary *extra = @{CRTableCell:cell};
                    [result appendAttributedString:[self attributedRuns:row[column][@"runs"] paragraph:p extra:extra]];
                    [result appendAttributedString:[[NSAttributedString alloc] initWithString:@"\n" attributes:[self visual:@{CRParagraph:p,CRTableCell:cell} paragraph:p]]];
                }
            }
            _emptyParagraph = CRBody();
            continue;
        }
        if (CRKind(p,@"math") || CRKind(p,@"image") || CRKind(p,@"diagram") || CRKind(p,@"toc") || CRKind(p,@"page")) {
            [result appendAttributedString:CRKind(p,@"toc") ? [self tocFigure:p] : CRKind(p,@"page") ? [self pageFigure:p] : [self figure:p]];
            if (i+1 < paragraphs.count) {
                NSMutableDictionary *ending=[[self visual:@{CRParagraph:p} paragraph:p] mutableCopy];
                [ending removeObjectForKey:NSBackgroundColorAttributeName];
                [result appendAttributedString:[[NSAttributedString alloc] initWithString:@"\n" attributes:ending]];
            }
            _emptyParagraph = CRBody();
            continue;
        }
        [result appendAttributedString:[self attributedRuns:p[@"runs"] paragraph:p extra:@{}]];
        if (i+1 < paragraphs.count) {
            // A newline carrying a chip background would paint through the
            // paragraph gap and merge adjacent opaque blocks into one slab.
            NSMutableDictionary *ending = [[self visual:@{CRParagraph:p} paragraph:p] mutableCopy];
            [ending removeObjectForKey:NSBackgroundColorAttributeName];
            [result appendAttributedString:[[NSAttributedString alloc] initWithString:@"\n" attributes:ending]];
        }
        _emptyParagraph = p;
    }
    _preparedMath=previousMath; _mathIndex=previousIndex;
    return result;
}
// normalizeScope is the edited text plus the caret, one paragraph either
// side, widened over any list, quote or footnote run it touches: those runs
// carry state (numbering, callout labels, continuation) from their start.
- (NSRange)normalizeScope {
    NSString *string=_text.string;
    NSUInteger length=string.length;
    NSRange seed=_dirty.location==NSNotFound ? _text.selectedRange : NSUnionRange(_dirty,_text.selectedRange);
    NSUInteger location=MIN(seed.location,length);
    seed=NSMakeRange(location,MIN(seed.length,length-location));
    NSRange range=[string paragraphRangeForRange:seed];
    BOOL (^inRun)(NSUInteger)=^BOOL(NSUInteger at) {
        NSDictionary *p=CRParagraphAt(self,at);
        return CRList(p) || CRKind(p,@"quote") || [p[@"footnoteGroup"] length]>0;
    };
    NSUInteger start=range.location, end=NSMaxRange(range);
    if (start) start=[string paragraphRangeForRange:NSMakeRange(start-1,0)].location;
    while (start) {
        NSUInteger previous=[string paragraphRangeForRange:NSMakeRange(start-1,0)].location;
        if (!inRun(previous)) break;
        start=previous;
    }
    if (end<length) end=NSMaxRange([string paragraphRangeForRange:NSMakeRange(end,0)]);
    while (end<length && inRun(end)) end=NSMaxRange([string paragraphRangeForRange:NSMakeRange(end,0)]);
    return NSMakeRange(start,end-start);
}
- (void)normalize {
    // NSTextView inherits attributes when a paragraph splits. Keep source
    // metadata once, then give newly created paragraphs fresh semantic identity.
    if (_mergeParagraph && _mergeLocation<=_text.string.length) {
        NSRange merged=[_text.string paragraphRangeForRange:NSMakeRange(_mergeLocation,0)];
        if (merged.length) { [_text.textStorage addAttribute:CRParagraph value:_mergeParagraph range:merged]; [self markDirty:merged]; }
    }
    _mergeParagraph=nil;
    NSRange scope=[self normalizeScope];
    [self refreshAlertsIn:scope];
    NSMutableSet *seen = [NSMutableSet set];
    NSString *previousFootnote=nil;
    for (NSValue *v in CRRangesIn(_text.string,scope)) {
        NSRange range = v.rangeValue;
        NSDictionary *p = CRParagraphAt(self,range.location), *original = p;
        if (CRKind(p,@"table")) { previousFootnote=p[@"footnoteGroup"]; continue; }
        if (CRStaleMarker(p,[_text.string substringWithRange:CRContent(_text.string,range)])) {
            // A rule or [TOC] edited into other text is a paragraph now (Typora).
            NSMutableDictionary *body=[CRBody() mutableCopy];
            if (p[@"separator"]) body[@"separator"]=p[@"separator"];
            p=body;
        }
        NSString *identifier = p[@"id"];
        if (identifier && [seen containsObject:identifier]) p = CRFresh(p);
        NSString *group=p[@"footnoteGroup"];
        if (group.length) {
            BOOL continuation=[previousFootnote isEqual:group];
            if ([p[@"footnoteContinuation"] boolValue]!=continuation) { NSMutableDictionary *updated=[p mutableCopy]; updated[@"footnoteContinuation"]=@(continuation); p=updated; }
        }
        previousFootnote=group;
        if (identifier) [seen addObject:identifier];
        if (p==original) continue; // untouched paragraphs keep their storage
        if (range.length) { [_text.textStorage addAttribute:CRParagraph value:p range:range]; [self markDirty:range]; }
        else { NSMutableDictionary *typing = [_text.typingAttributes mutableCopy]; typing[CRParagraph]=p; _text.typingAttributes=typing; _emptyParagraph=p; }
    }
    [self renumberIn:scope]; // deleting or merging items must not leave gaps
    [self restyleDirty];
}
- (NSDictionary *)document { return [self documentForRange:NSMakeRange(NSNotFound,0)]; }
- (NSDictionary *)documentForRange:(NSRange)selection {
    NSMutableArray *paragraphs = [NSMutableArray array];
    NSMutableSet *sourceFigures=[NSMutableSet set];
    NSTextTable *currentTable = nil;
    NSMutableDictionary *tableParagraph = nil;
    for (NSValue *v in CRRanges(_text.string)) {
        NSRange range = CRContent(_text.string,v.rangeValue);
        if (range.length == 0 && range.location == _text.string.length && range.location > 0 && CRCellAt(self,range.location-1)) continue;
        NSTextTableBlock *cell = CRCellAt(self,range.location);
        if (selection.location != NSNotFound) {
            BOOL selectedEmptyCell=NO;
            if (cell && !range.length) {
                NSRange tableRange=CRTableRangeNear(self,cell.table,range.location);
                selectedEmptyCell=selection.location<=tableRange.location && NSMaxRange(selection)>=NSMaxRange(tableRange);
            }
            range=NSIntersectionRange(range,selection);
            if (!range.length && !selectedEmptyCell) continue;
        }
        NSDictionary *sourceFigure=range.location<_text.string.length ? [_text.textStorage attribute:CRFootnoteFigureSource atIndex:range.location effectiveRange:NULL] : nil;
        if (sourceFigure) {
            NSRange expanded;
            [_text.textStorage attribute:CRFootnoteFigureSource atIndex:range.location longestEffectiveRange:&expanded inRange:NSMakeRange(0,_text.string.length)];
            if ((selection.location==NSNotFound || NSIntersectionRange(selection,expanded).length==expanded.length) && [[_text.string substringWithRange:expanded] isEqual:sourceFigure[@"source"]]) {
                if (![sourceFigures containsObject:sourceFigure[@"id"]]) { [paragraphs addObject:sourceFigure[@"paragraph"]]; [sourceFigures addObject:sourceFigure[@"id"]]; }
                continue;
            }
        }
        NSMutableDictionary *p = [CRParagraphAt(self,range.location) mutableCopy];
        BOOL figure=CRKind(p,@"math") || CRKind(p,@"image") || CRKind(p,@"diagram") || CRKind(p,@"toc") || CRKind(p,@"page");
        if (figure && ![[_text.string substringWithRange:range] isEqual:@"\uFFFC"]) {
            // The user typed into the figure: it is source text now. With the
            // figure itself deleted, what is left is prose.
            p[@"kind"]=[[_text.string substringWithRange:range] containsString:@"\uFFFC"] ? @"source" : @"body";
            if (CRKind(p,@"body")) [p removeObjectForKey:@"raw"];
            figure=NO;
        }
        NSString *sourceRaw=[p[@"raw"] isKindOfClass:NSString.class] ? p[@"raw"] : @"";
        if (CRStaleMarker(p,[_text.string substringWithRange:range]) || (CRKind(p,@"source") && [sourceRaw containsString:@"\n"] && ![sourceRaw isEqual:[_text.string substringWithRange:range]])) {
            // An edited rule that is no longer one ("--") is prose (Typora),
            // as is the line left of a deleted multi-line block.
            p[@"kind"]=@"body"; [p removeObjectForKey:@"raw"];
        }
        if (!CRKind(p,@"opaque") && !figure) {
            NSMutableArray *runs = [NSMutableArray array];
            [_text.textStorage enumerateAttributesInRange:range options:0 usingBlock:^(NSDictionary *attrs, NSRange r, BOOL *stop) {
                if (attrs[CRInlineSource]) {
                    id origin=attrs[CRInlineSource];
                    if ([origin isKindOfClass:NSDictionary.class]) {
                        NSRange expanded;
                        [self.text.textStorage attribute:CRInlineSource atIndex:r.location longestEffectiveRange:&expanded inRange:NSMakeRange(0,self.text.string.length)];
                        if (NSIntersectionRange(range,expanded).length==expanded.length && [[self.text.string substringWithRange:expanded] isEqual:origin[@"source"]]) {
                            if (r.location==expanded.location) [runs addObjectsFromArray:origin[@"runs"] ?: @[]];
                            return;
                        }
                    }
                    NSString *source=[self.text.string substringWithRange:r];
                    if ([runs.lastObject[@"source"] boolValue]) {
                        NSString *previous=runs.lastObject[@"text"];
                        [runs removeLastObject];
                        source=[previous stringByAppendingString:source];
                    }
                    [runs addObject:@{@"text":source,@"source":@YES}];
                    return;
                }
                if (attrs[CRMathText]) {
                    [runs addObject:@{@"text":attrs[CRMathText],@"math":attrs[CRMathExpr]}];
                    return;
                }
                if (attrs[CRImageSource] && attrs[NSAttachmentAttributeName]) {
                    NSMutableDictionary *image=[@{@"text":attrs[CRImageSource],@"image":attrs[CRImageURL] ?: @""} mutableCopy];
                    id link=attrs[NSLinkAttributeName];
                    if (link) image[@"link"]=[link isKindOfClass:NSURL.class] ? [link absoluteString] : link;
                    if (attrs[CRLinkTitle]) image[@"linkTitle"]=attrs[CRLinkTitle];
                    for (NSUInteger i=0;i<r.length;i++) [runs addObject:image];
                    return;
                }
                NSDictionary *footnote=attrs[CRFootnote];
                NSString *chunk=footnote ? footnote[@"text"] : [[self.text.string substringWithRange:r] stringByReplacingOccurrencesOfString:@"\uFFFC" withString:@""];
                if (!chunk.length) return;
                NSMutableDictionary *base = [NSMutableDictionary dictionary];
                if (footnote) base[@"footnote"]=footnote[@"footnote"];
                if ([attrs[CRBold] boolValue]) base[@"bold"]=@YES;
                if ([attrs[CRItalic] boolValue]) base[@"italic"]=@YES;
                if ([attrs[CRCode] boolValue]) base[@"code"]=@YES;
                if ([attrs[NSStrikethroughStyleAttributeName] integerValue]) base[@"strike"]=@YES;
                if ([attrs[CRUnderline] boolValue]) base[@"underline"]=@YES;
                if ([attrs[CRKbd] boolValue]) base[@"kbd"]=@YES;
                if ([attrs[CRHighlight] boolValue]) base[@"highlight"]=@YES;
                if ([attrs[CRHTML] boolValue]) base[@"html"]=@YES;
                if ([attrs[CRSup] boolValue]) base[@"sup"]=@YES;
                if ([attrs[CRSub] boolValue]) base[@"sub"]=@YES;
                if ([attrs[CRColor] isKindOfClass:NSString.class] && [attrs[CRColor] length]) base[@"color"]=attrs[CRColor];
                if ([attrs[CRBackground] isKindOfClass:NSString.class] && [attrs[CRBackground] length]) base[@"background"]=attrs[CRBackground];
                if (attrs[NSLinkAttributeName]) base[@"link"]=[attrs[NSLinkAttributeName] description];
                if (attrs[NSLinkAttributeName] && attrs[CRLinkTitle]) base[@"linkTitle"]=attrs[CRLinkTitle];
                // U+2028 line separators become break runs so a soft visual
                // line survives the round-trip as a Markdown hard break.
                // Inside a tag a line separator is the tag's own newline.
                NSArray *pieces=[attrs[CRHTML] boolValue] ? @[[chunk stringByReplacingOccurrencesOfString:@"\u2028" withString:@"\n"]] : [chunk componentsSeparatedByString:@"\u2028"];
                for (NSUInteger pi=0; pi<pieces.count; pi++) {
                    if (pi) [runs addObject:[attrs[CRSoftBreak] boolValue] ? @{@"break":@YES,@"soft":@YES} : @{@"break":@YES}];
                    NSString *piece=pieces[pi];
                    if (!piece.length) continue;
                    NSMutableDictionary *run=[base mutableCopy];
                    run[@"text"]=piece;
                    [runs addObject:run];
                }
            }];
            p[@"runs"] = runs;
        }
        if (cell) {
            NSRange tableRange = CRTableRangeNear(self,cell.table,range.location);
            BOOL whole = selection.location == NSNotFound || (selection.location <= tableRange.location && NSMaxRange(selection) >= NSMaxRange(tableRange));
            if (!whole) { [paragraphs addObject:@{@"kind":@"body", @"runs":p[@"runs"]}]; continue; }
            if (currentTable != cell.table) {
                currentTable = cell.table;
                tableParagraph = [p mutableCopy]; tableParagraph[@"runs"] = @[];
                NSMutableArray *rows = [NSMutableArray array];
                for (NSArray *originalRow in p[@"rows"]) {
                    NSMutableArray *row = [NSMutableArray array];
                    for (NSDictionary *originalCell in originalRow) [row addObject:@{@"runs":@[]}];
                    [rows addObject:row];
                }
                tableParagraph[@"rows"] = rows;
                [paragraphs addObject:tableParagraph];
            }
            // A cell met twice (text an edit left carrying the cell) keeps
            // both parts rather than losing the first.
            NSArray *previous=tableParagraph[@"rows"][cell.startingRow][cell.startingColumn][@"runs"];
            NSArray *runs=p[@"runs"];
            if ([previous count] && [runs count]) runs=[[previous arrayByAddingObject:@{@"break":@YES}] arrayByAddingObjectsFromArray:runs];
            else if ([previous count]) runs=previous;
            tableParagraph[@"rows"][cell.startingRow][cell.startingColumn] = @{@"runs":runs};
            continue;
        }
        currentTable = nil;
        if (CRKind(p,@"table")) {
            // A line that inherited a table's metadata outside any cell is
            // text; its stale rows would write a second copy of the table.
            NSMutableDictionary *body=[CRBody() mutableCopy];
            body[@"runs"]=p[@"runs"] ?: @[];
            if (p[@"separator"]) body[@"separator"]=p[@"separator"];
            p=body;
        }
        [paragraphs addObject:p];
    }
    return @{@"paragraphs":paragraphs};
}
- (BOOL)allowRange:(NSRange)range {
    if (![self allowFootnoteRange:range]) return NO;
    if (range.length) {
        for (NSValue *value in CRRanges(_text.string)) {
            NSRange paragraph = value.rangeValue;
            NSTextTableBlock *cell = CRCellAt(self,paragraph.location);
            if (!cell) continue;
            BOOL removesBoundary = NSLocationInRange(NSMaxRange(paragraph)-1,range) || (paragraph.location && NSLocationInRange(paragraph.location-1,range));
            if (removesBoundary) {
                NSRange tableRange = CRTableRange(self,cell.table);
                if (range.location > tableRange.location || NSMaxRange(range) < NSMaxRange(tableRange)) { NSBeep(); return NO; }
            }
        }
    }
    for (NSValue *v in CRRanges(_text.string)) {
        NSRange r = CRContent(_text.string,v.rangeValue);
        if (!CRKind(CRParagraphAt(self,r.location),@"opaque")) continue;
        if (range.length && r.location > 0 && NSLocationInRange(r.location-1,range) && NSMaxRange(range) < NSMaxRange(r)) { [self source]; return NO; }
        if (range.length && NSMaxRange(r)<_text.string.length && NSLocationInRange(NSMaxRange(r),range) && range.location>r.location) { [self source]; return NO; }
        BOOL intersects = range.length ? NSIntersectionRange(range,r).length > 0 : range.location >= r.location && range.location <= NSMaxRange(r);
        if (intersects && !(range.length && range.location <= r.location && NSMaxRange(range)>=NSMaxRange(r))) { [self source]; return NO; }
    }
    return YES;
}
- (NSDictionary *)textView:(NSTextView *)view shouldChangeTypingAttributes:(NSDictionary *)old toAttributes:(NSDictionary *)attrs {
    if (attrs[CRImageSource]) {
        // Text typed beside an inline image is text, not another image.
        NSMutableDictionary *plain=[attrs mutableCopy];
        for (NSString *key in @[CRImageSource,CRImageURL,NSAttachmentAttributeName]) [plain removeObjectForKey:key];
        attrs=plain;
    }
    if (attrs[CRMathText] || attrs[CRFootnote]) attrs=CRTextAttrs(attrs); // text typed after a deleted formula or footnote mark
    // Text typed beside an HTML tag is prose, not more markup.
    if (!attrs[CRHTML]) return attrs;
    NSUInteger at=view.selectedRange.location;
    if (at>0 && at<view.string.length && [view.textStorage attribute:CRHTML atIndex:at effectiveRange:NULL] && [view.textStorage attribute:CRHTML atIndex:at-1 effectiveRange:NULL]) return attrs; // editing inside the tag
    NSMutableDictionary *plain=[attrs mutableCopy]; [plain removeObjectForKey:CRHTML];
    return [self visual:plain paragraph:plain[CRParagraph] ?: CRBody()];
}
- (BOOL)textView:(NSTextView *)textView shouldChangeTextInRange:(NSRange)range replacementString:(NSString *)text {
    BOOL allowed=_suppress || [self allowRange:range];
    if (allowed) [self markDirty:NSMakeRange(range.location,text.length)];
    // Deleting across blocks keeps the block the selection started in
    // (Typora): "1. a|\n2. b" stays item 1, not item 2.
    _mergeParagraph=nil;
    if (allowed && !_suppress && range.length && NSMaxRange(range)<=_text.string.length && [[_text.string substringWithRange:range] rangeOfCharacterFromSet:NSCharacterSet.newlineCharacterSet].location!=NSNotFound) {
        NSDictionary *start=CRParagraphAt(self,range.location);
        BOOL guarded=CRCellAt(self,range.location) || CRCellAt(self,NSMaxRange(range)) || CRKind(start,@"code") || CRKind(start,@"source") || CRKind(start,@"opaque") || CRFigure(start) || [start[@"footnoteId"] length];
        if (!guarded && NSMaxRange(range)<_text.string.length) {
            // The joined block ends where the absorbed one did: it takes that
            // block's trailing separator, so the file keeps its ending.
            NSDictionary *end=CRParagraphAt(self,NSMaxRange(range));
            NSMutableDictionary *merged=[start mutableCopy];
            for (NSString *key in @[@"separator",@"nextId"]) { if (end[key]) merged[key]=end[key]; else [merged removeObjectForKey:key]; }
            _mergeParagraph=merged; _mergeLocation=range.location;
        }
    }
    return allowed;
}
- (void)textDidChange:(NSNotification *)notification {
    [_text clearHover];
    if (_suppress) return;
    if (!_text.hasMarkedText) [self normalize];
    if (_typewriter) [self centerCaretLine];
    [self notify];
}
- (void)textViewDidChangeSelection:(NSNotification *)notification {
    [_text clearHover];
    [self slashQueryChanged];
    [self emojiMenuQueryChanged];
    [self linkMenuQueryChanged];
    [self updateFormatBar];
    [self updateTypingParagraph];
    [self finishDraftGroup];
    [self finishInlineSource:NO];
    if (_suppress || _text.hasMarkedText) return;
    // A rule or callout marker shows its source only while the caret is on
    // it, so just the paragraphs the caret left and entered change.
    NSRange caret=[_text.string paragraphRangeForRange:_text.selectedRange];
    [self refreshRulesIn:_ruleCaret.location==NSNotFound || NSMaxRange(_ruleCaret)>_text.string.length ? caret : NSUnionRange(caret,_ruleCaret)];
    _ruleCaret=caret;
    [_text setNeedsDisplay:YES];
    if (_focusMode) [self restyle]; // the dimmed paragraphs follow the caret
    if (_typewriter) [self centerCaretLine];
}
- (void)textDidEndEditing:(NSNotification *)notification {
    // Leaving the editor finishes any open draft group: the block the caret was
    // editing re-renders to its figure form (a formula/image/code), regardless
    // of where the caret sat inside it.
    if (_suppress || _text.hasMarkedText) return;
    [self slashClose];
    [self emojiMenuClose];
    [self linkMenuClose];
    [self formatBarHide];
    NSRange group=_draftGroup;
    _draftGroup=NSMakeRange(NSNotFound,0);
    if (group.location!=NSNotFound) [self renderDraftGroup:group];
    [self finishInlineSource:YES];
}
- (void)updateTypingParagraph {
    if (_suppress || _text.hasMarkedText || _text.selectedRange.length) return;
    NSUInteger location = _text.selectedRange.location;
    NSDictionary *p = CRParagraphAt(self,location);
    NSMutableDictionary *typing = [_text.typingAttributes mutableCopy];
    id expanded=location<_text.string.length ? [_text.textStorage attribute:CRInlineSource atIndex:location effectiveRange:NULL] : nil;
    if (expanded) typing[CRInlineSource]=expanded;
    else [typing removeObjectForKey:CRInlineSource];
    for (NSString *key in @[CRMathText,CRMathExpr,CRMathNumber,CRMathRendered,CRFootnote,CRFootnoteNumber,CRSoftBreak,NSToolTipAttributeName,NSAttachmentAttributeName]) [typing removeObjectForKey:key];
    typing[CRParagraph] = p;
    NSTextTableBlock *cell = CRCellAt(self,location);
    if (cell) typing[CRTableCell] = cell; else [typing removeObjectForKey:CRTableCell];
    _text.typingAttributes = [self visual:typing paragraph:p];
}
- (void)restore:(NSAttributedString *)value selection:(NSRange)selection typing:(NSDictionary *)typing ending:(NSDictionary *)ending {
    NSAttributedString *before = [_text.textStorage copy]; NSRange oldSelection = _text.selectedRange; NSDictionary *oldTyping = _text.typingAttributes; NSDictionary *oldEnding=_emptyParagraph;
    [_undo registerUndoWithTarget:self handler:^(CREditor *target) { [target restore:before selection:oldSelection typing:oldTyping ending:oldEnding]; }];
    _suppress=YES;
    // Undo swaps the whole storage; only the span that differs needs style.
    NSString *old=_text.string, *now=value.string;
    NSUInteger head=0, tail=0, shortest=MIN(old.length,now.length);
    while (head<shortest && [old characterAtIndex:head]==[now characterAtIndex:head]) head++;
    while (tail<shortest-head && [old characterAtIndex:old.length-1-tail]==[now characterAtIndex:now.length-1-tail]) tail++;
    // Swap only the differing paragraphs when no table is involved: a full
    // swap relays out and re-runs font fallback over the whole note.
    NSRange oldSpan=[old paragraphRangeForRange:NSMakeRange(head,old.length-head-tail)];
    NSUInteger spanTail=old.length-NSMaxRange(oldSpan);
    // Tables are swapped whole: an NSTextTable must not be split between the
    // old and the restored storage.
    NSAttributedString *current=_text.textStorage;
    NSUInteger spanHead=oldSpan.location;
    for (int pass=0; pass<4; pass++) {
        NSUInteger head0=spanHead, tail0=spanTail;
        for (NSAttributedString *text in @[current,value]) {
            if (text.length<spanHead+spanTail) continue;
            NSRange r=NSMakeRange(spanHead,text.length-spanTail-spanHead);
            NSRange covered=CRTableSpan(text,r);
            if (covered.location==NSNotFound) continue;
            covered=[text.string paragraphRangeForRange:covered];
            spanHead=MIN(spanHead,covered.location);
            spanTail=MIN(spanTail,text.length-NSMaxRange(covered));
        }
        if (head0==spanHead && tail0==spanTail) break;
    }
    oldSpan=NSMakeRange(spanHead,old.length-spanTail-spanHead);
    NSUInteger newHead=MIN(spanHead,now.length);
    BOOL fits=now.length>=spanTail+newHead && old.length>=spanTail+spanHead;
    NSRange newSpan=NSMakeRange(newHead,fits ? now.length-spanTail-newHead : 0);
    BOOL partial=fits && newHead==oldSpan.location;
    if (partial) {
        [_text.textStorage beginEditing];
        [_text.textStorage replaceCharactersInRange:oldSpan withAttributedString:[value attributedSubstringFromRange:newSpan]];
        // Structural undo can also restore attributes outside the text that
        // changed (list numbering, continuation flags).
        [value enumerateAttributesInRange:NSMakeRange(0,value.length) options:0 usingBlock:^(NSDictionary *attrs, NSRange r, BOOL *stop) {
            if (NSLocationInRange(r.location,newSpan) && NSMaxRange(r)<=NSMaxRange(newSpan)) return;
            NSRange effective;
            NSDictionary *existing=[self.text.textStorage attributesAtIndex:r.location effectiveRange:&effective];
            if (NSMaxRange(effective)>=NSMaxRange(r) && effective.location<=r.location && [existing isEqualToDictionary:attrs]) return;
            [self.text.textStorage setAttributes:attrs range:r];
        }];
        [_text.textStorage endEditing];
    } else [_text.textStorage setAttributedString:value];
    _text.selectedRange=selection; _text.typingAttributes=typing; _emptyParagraph=ending;
    _suppress=NO;
    [self markDirty:NSMakeRange(head,now.length-head-tail)];
    [self restyleDirty]; [self notify];
}
- (void)replace:(NSRange)range with:(NSAttributedString *)value selection:(NSRange)selection {
    if (!_text.editable || _text.hasMarkedText || ![self allowRange:range]) return;
    [_text breakUndoCoalescing];
    NSAttributedString *before=[_text.textStorage copy]; NSRange oldSelection=_text.selectedRange; NSDictionary *oldTyping=_text.typingAttributes; NSDictionary *oldEnding=_emptyParagraph;
    [_undo beginUndoGrouping];
    [_undo registerUndoWithTarget:self handler:^(CREditor *target) { [target restore:before selection:oldSelection typing:oldTyping ending:oldEnding]; }];
    BOOL wasSuppress=_suppress; _suppress=YES;
    [_text.textStorage replaceCharactersInRange:range withAttributedString:value]; _text.selectedRange=selection;
    [self markDirty:NSMakeRange(range.location,value.length)];
    _suppress=wasSuppress;
    [self normalize];
    [_undo endUndoGrouping];
    [self notify];
}
// applyColor sets ("color:#hex", "background:#hex") or clears ("color:",
// "background:") a text or background color on the selection, or on what is
// typed next when nothing is selected.
- (void)applyColor:(NSString *)command selection:(NSRange)selection {
    BOOL background=[command hasPrefix:@"background:"];
    NSString *key=background ? CRBackground : CRColor;
    NSString *value=[[command substringFromIndex:background ? 11 : 6] lowercaseString];
    if (value.length && !CRResolvedColor(value,background,NO)) return;
    if (!selection.length) {
        NSMutableDictionary *typing=[_text.typingAttributes mutableCopy];
        if (value.length) typing[key]=value; else [typing removeObjectForKey:key];
        _text.typingAttributes=[self visual:typing paragraph:typing[CRParagraph] ?: CRBody()];
        return;
    }
    if (NSMaxRange(selection)>_text.string.length) return;
    NSMutableAttributedString *after=[_text.textStorage mutableCopy];
    BOOL changed=NO;
    for (NSValue *v in CRSelectedParagraphs(self)) {
        NSDictionary *p=CRParagraphAt(self,v.rangeValue.location);
        // Figures and source lines have no inline styles to color.
        if ([@[@"code",@"source",@"opaque",@"math",@"image",@"diagram",@"toc",@"page"] containsObject:p[@"kind"] ?: @""]) continue;
        // Line endings stay uncolored: a tinted newline paints a stray chip.
        NSRange part=NSIntersectionRange(CRContent(_text.string,v.rangeValue),selection);
        if (!part.length) continue;
        if (value.length) [after addAttribute:key value:value range:part];
        else [after removeAttribute:key range:part];
        changed=YES;
    }
    if (!changed) return;
    BOOL wasSuppress=_suppress; _suppress=YES;
    [self replace:NSMakeRange(0,_text.string.length) with:after selection:selection];
    _suppress=wasSuppress; [self notify];
}
- (void)format:(NSString *)command {
    if ([command isEqual:@"footnote:insert"]) { [self insertFootnote]; return; }
    if ([command isEqual:@"footnote:definition"] || [command isEqual:@"footnote:reference"]) {
        [self navigateFootnote:[command isEqual:@"footnote:definition"] atLocation:_text.selectedRange.location]; return;
    }
    if ([command isEqual:@"footnote:back"]) { [self footnoteNavigateBack]; return; }
    if ([command hasPrefix:@"code-line-numbers:"]) {
        BOOL enabled=[[command substringFromIndex:18] boolValue];
        if (_codeLineNumbers!=enabled) { _codeLineNumbers=enabled; [self restyle]; }
        return;
    }
    if ([command hasPrefix:@"code-language:"]) {
        [self setCodeLanguage:[command substringFromIndex:14]];
        return;
    }
    if ([command hasPrefix:@"quote-alert:"]) { [self setQuoteAlert:[command substringFromIndex:12]]; return; }
    if ([command hasPrefix:@"image-size:"]) { [self setImageSize:[command substringFromIndex:11]]; return; }
    if ([command hasPrefix:@"math-numbering:"]) {
        if (_text.hasMarkedText) return;
        NSString *mode=[command substringFromIndex:15];
        if ([@[@"none",@"ams",@"all"] containsObject:mode] && ![_mathNumbering isEqual:mode]) { _mathNumbering=mode; [self refreshMath]; }
        return;
    }
    if ([command isEqual:@"math-refresh"]) { [self refreshAllMath]; return; }
    if (!_text.editable || _text.hasMarkedText) return;
    [self finishInlineSource:YES];
    if ([command isEqual:@"block-up"]) { [self moveBlock:NO]; return; }
    if ([command isEqual:@"block-down"]) { [self moveBlock:YES]; return; }
    if ([command isEqual:@"block-duplicate"]) { [self duplicateBlock]; return; }
    if ([command isEqual:@"block-delete"]) { [self deleteBlock]; return; }
    if ([command hasPrefix:@"table:"]) { [self tableCommand:command]; return; }
    if ([command hasPrefix:@"insert:"]) { [self insertBlock:[command substringFromIndex:7]]; return; }
    if ([command isEqual:@"list-paragraph"]) { [self continueListParagraph]; return; }
    if ([command isEqual:@"indent"] || [command isEqual:@"outdent"]) {
        BOOL outdent=[command isEqual:@"outdent"];
        if (![self indentCode:outdent] && ![self indentQuote:outdent]) [self indentList:outdent];
        return;
    }
    NSRange selection = _text.selectedRange;
    for (NSValue *v in CRSelectedParagraphs(self)) if (CRKind(CRParagraphAt(self,v.rangeValue.location),@"opaque")) { [self source]; return; }
    if ([command hasPrefix:@"link:"]) {
        // link:URL wraps the selection in a link run; "link:URL\nTEXT" with
        // no selection inserts TEXT (or the URL) as a link at the caret.
        NSString *url=[command substringFromIndex:5], *label=nil;
        NSRange split=[url rangeOfString:@"\n"];
        if (split.location!=NSNotFound) { label=[url substringFromIndex:NSMaxRange(split)]; url=[url substringToIndex:split.location]; }
        if (!url.length) return;
        if (!selection.length) {
            if (CRCellAt(self,selection.location) || CRKind(CRParagraphAt(self,selection.location),@"code") || CRKind(CRParagraphAt(self,selection.location),@"source")) return;
            NSString *text=label.length ? label : url;
            NSMutableDictionary *typing=[_text.typingAttributes mutableCopy];
            for (NSString *key in @[CRInlineSource,CRMathText,CRMathExpr,CRMathNumber,CRMathRendered,NSAttachmentAttributeName,CRCode]) [typing removeObjectForKey:key];
            typing[NSLinkAttributeName]=url;
            NSDictionary *paragraph=CRParagraphAt(self,selection.location);
            NSAttributedString *value=[[NSAttributedString alloc] initWithString:text attributes:[self visual:typing paragraph:paragraph]];
            NSDictionary *after=_text.typingAttributes;
            [self replace:selection with:value selection:NSMakeRange(selection.location+text.length,0)];
            NSMutableDictionary *plain=[after mutableCopy]; [plain removeObjectForKey:NSLinkAttributeName];
            _text.typingAttributes=plain; // keep typing outside the link
            return;
        }
        NSMutableAttributedString *after=[_text.textStorage mutableCopy];
        [after addAttribute:NSLinkAttributeName value:url range:selection];
        [self replace:NSMakeRange(0,_text.string.length) with:after selection:selection];
        return;
    }
    if ([command hasPrefix:@"color:"] || [command hasPrefix:@"background:"]) { [self applyColor:command selection:selection]; return; }
    NSString *key = @{@"bold":CRBold,@"italic":CRItalic,@"strike":NSStrikethroughStyleAttributeName,@"code":CRCode,@"underline":CRUnderline,@"highlight":CRHighlight,@"sup":CRSup,@"sub":CRSub}[command];
    BOOL clear = [command isEqual:@"clear"];
    if (!key) for (NSValue *value in CRSelectedParagraphs(self)) if (CRKind(CRParagraphAt(self,value.rangeValue.location),@"table")) return;
    if (key && selection.length == 0) {
        NSMutableDictionary *typing=[_text.typingAttributes mutableCopy];
        if ([typing[key] boolValue]) [typing removeObjectForKey:key];
        else {
            typing[key]=@YES;
            if (key==CRSup) [typing removeObjectForKey:CRSub]; // sup and sub exclude each other
            if (key==CRSub) [typing removeObjectForKey:CRSup];
        }
        _text.typingAttributes=[self visual:typing paragraph:typing[CRParagraph] ?: CRBody()];
        return;
    }
    NSMutableAttributedString *after=[_text.textStorage mutableCopy];
    NSDictionary *newTyping=nil;
    NSString *commandList=NSUUID.UUID.UUIDString;
    if (key) {
        BOOL enabled=YES;
        NSUInteger end=NSMaxRange(selection);
        for (NSUInteger i=selection.location;i<end;i++) if (![[after attribute:key atIndex:i effectiveRange:NULL] boolValue]) { enabled=NO; break; }
        if (enabled) [after removeAttribute:key range:selection];
        else {
            [after addAttribute:key value:@YES range:selection];
            if (key==CRSup) [after removeAttribute:CRSub range:selection];
            if (key==CRSub) [after removeAttribute:CRSup range:selection];
        }
    } else {
        NSString *kind = @{@"body":@"body",@"bullet":@"bullet",@"ordered":@"ordered",@"checklist":@"checklist",@"quote":@"quote"}[command];
        NSInteger level=0;
        if ([command hasPrefix:@"heading"] && command.length==8) {
            unichar digit=[command characterAtIndex:7];
            if (digit>='1' && digit<='6') { kind=@"heading"; level=digit-'0'; }
        }
        if (clear) kind=@"body";
        if (!kind) return;
        if (!clear && ![kind isEqual:@"body"]) {
            // The same block command again turns the lines back into body
            // text (Typora): Cmd+2 on an h2, the list button on a list.
            BOOL same=YES;
            for (NSValue *v in CRSelectedParagraphs(self)) {
                NSDictionary *p=CRParagraphAt(self,v.rangeValue.location);
                BOOL match=[kind isEqual:@"heading"] ? CRKind(p,@"heading") && [p[@"level"] integerValue]==level
                    : [kind isEqual:@"ordered"] ? CRKind(p,@"ordered") : CRKind(p,kind) && !([kind isEqual:@"checklist"] && CROrdered(p));
                same &= match;
            }
            if (same) kind=@"body";
        }
        NSArray *selected=CRSelectedParagraphs(self);
        for (NSValue *v in selected) {
            NSRange r=v.rangeValue;
            NSMutableDictionary *p=[CRParagraphAt(self,r.location) mutableCopy];
            if (CRKind(p,@"source")) continue; // raw HTML and the like stay raw
            // Lines joined into one quote or list lose their old blank-line
            // separators, so they serialize as one block, not several.
            if (v!=selected.lastObject && ([kind isEqual:@"quote"] || [@[@"bullet",@"ordered",@"checklist"] containsObject:kind])) [p removeObjectForKey:@"separator"];
            p[@"kind"]=kind;
            [p removeObjectForKey:@"level"]; [p removeObjectForKey:@"checked"]; [p removeObjectForKey:@"ordinal"];
            if (![@[@"bullet",@"ordered",@"checklist"] containsObject:kind]) {
                for (NSString *field in @[@"depth",@"listId",@"listIndent",@"listBaseline",@"listContainer",@"listMarker",@"listItem",@"listContinuation"]) [p removeObjectForKey:field];
            } else {
                // A quote's depth counts quotes, not list levels.
                // (and a quote or code line inside an item starts a list of its own).
                if (!CRList(CRParagraphAt(self,r.location))) for (NSString *field in @[@"depth",@"codeIndent",@"itemDepth",@"listContinuation",@"listItem",@"listId",@"listContainer",@"listMarker",@"listIndent",@"listBaseline"]) [p removeObjectForKey:field];
                if (!p[@"listItem"]) p[@"listItem"]=NSUUID.UUID.UUIDString;
                if (!p[@"listId"]) p[@"listId"]=commandList;
                if (!p[@"listContainer"]) p[@"listContainer"]=commandList;
            }
            if ([kind isEqual:@"heading"]) {
                // Headings cannot sit inside a list item or a quote here.
                p[@"level"]=@(level);
                for (NSString *field in @[@"quote",@"codeIndent",@"itemDepth"]) [p removeObjectForKey:field];
            }
            if ([kind isEqual:@"ordered"]) p[@"ordinal"]=@1;
            if (r.length) [after addAttribute:CRParagraph value:p range:r];
            else { NSMutableDictionary *typing=[_text.typingAttributes mutableCopy]; typing[CRParagraph]=p; newTyping=[self visual:typing paragraph:p]; }
        }
        if (clear) {
            // Clear strips the inline traits as well as the paragraph kind.
            NSArray *traits=@[CRBold,CRItalic,CRCode,CRUnderline,CRKbd,CRHighlight,CRSup,CRSub,CRColor,CRBackground,NSStrikethroughStyleAttributeName,NSLinkAttributeName,CRLinkTitle];
            if (selection.length) for (NSString *trait in traits) [after removeAttribute:trait range:selection];
            else if (!CRKind(CRParagraphAt(self,selection.location),@"source")) {
                NSMutableDictionary *p=[CRParagraphAt(self,selection.location) mutableCopy];
                p[@"kind"]=@"body"; [p removeObjectForKey:@"level"]; [p removeObjectForKey:@"checked"]; [p removeObjectForKey:@"ordinal"];
                for (NSString *field in @[@"depth",@"listId",@"listIndent",@"listBaseline",@"listContainer",@"listMarker",@"listItem",@"listContinuation"]) [p removeObjectForKey:field];
                NSMutableDictionary *typing=[_text.typingAttributes mutableCopy];
                for (NSString *trait in traits) [typing removeObjectForKey:trait];
                typing[CRParagraph]=p;
                newTyping=[self visual:typing paragraph:p];
            }
        }
    }
    BOOL wasSuppress=_suppress; _suppress=YES;
    [self replace:NSMakeRange(0,_text.string.length) with:after selection:selection];
    if (newTyping) { _text.typingAttributes=newTyping; _emptyParagraph=newTyping[CRParagraph]; }
    if (!key) [self renumber];
    _suppress=wasSuppress; [self notify];
    [self updateFormatBar];
}
// leaveQuotedList turns a top-level list item inside a quote back into a
// line of that quote (Enter on an empty item, Backspace at its start).
- (BOOL)leaveQuotedList:(NSRange)paragraph {
    NSDictionary *p=CRParagraphAt(self,paragraph.location);
    if (!CRList(p) || CRQuoteOf(p)==0 || [p[@"depth"] integerValue]>0) return NO;
    NSMutableDictionary *quote=[@{@"kind":@"quote",@"runs":@[],@"depth":@(CRQuoteOf(p)-1)} mutableCopy];
    if ([p[@"alert"] length]) quote[@"alert"]=p[@"alert"];
    if (p[@"separator"]) quote[@"separator"]=p[@"separator"];
    NSMutableAttributedString *after=[_text.textStorage mutableCopy];
    NSRange content=CRContent(_text.string,paragraph);
    NSRange whole=NSMakeRange(paragraph.location,MIN(paragraph.length,after.length-paragraph.location));
    [_text.textStorage enumerateAttributesInRange:whole options:0 usingBlock:^(NSDictionary *attrs, NSRange r, BOOL *stop) {
        NSMutableDictionary *a=[attrs mutableCopy]; a[CRParagraph]=quote;
        [after setAttributes:[self visual:a paragraph:quote] range:r];
    }];
    NSDictionary *typing=[self visual:@{CRParagraph:quote} paragraph:quote];
    BOOL ending=paragraph.location>=_text.string.length || (content.length==0 && NSMaxRange(paragraph)>=_text.string.length && ![_text.string hasSuffix:@"\n"]);
    [self beginStructuralCommand]; [_undo beginUndoGrouping];
    [self restore:after selection:_text.selectedRange typing:typing ending:ending ? quote : _emptyParagraph];
    [_undo endUndoGrouping];
    return YES;
}
- (void)newline {
    if (!_text.editable || _text.hasMarkedText) { [_text insertText:@"\n" replacementRange:_text.selectedRange]; return; }
    [self emojiRule:NO]; // Enter closes a trailing :emoji: before the split
    NSRange selection=_text.selectedRange;
    if (CRCellAt(self,selection.location)) { [self moveTableCell:NO]; return; }
    NSRange paragraph=[_text.string paragraphRangeForRange:NSMakeRange(selection.location,0)];
    NSDictionary *p=CRParagraphAt(self,paragraph.location);
    if (CRKind(p,@"code")) { [self codeNewline]; return; }
    BOOL emptyFootnote=[p[@"footnoteId"] length] && CRContent(_text.string,paragraph).length==0;
    if (emptyFootnote && [p[@"footnoteContinuation"] boolValue]) {
        NSMutableAttributedString *after=[_text.textStorage mutableCopy];
        NSDictionary *body=CRBody(), *attrs=[self visual:@{CRParagraph:body} paragraph:body];
        if (paragraph.length) [after setAttributes:attrs range:paragraph];
        [self beginStructuralCommand]; [_undo beginUndoGrouping];
        [self restore:after selection:selection typing:attrs ending:paragraph.location==_text.string.length ? body : _emptyParagraph];
        [_undo endUndoGrouping]; return;
    }
    BOOL list=CRKind(p,@"checklist") || CRKind(p,@"bullet") || CRKind(p,@"ordered");
    // Enter on an empty list item or quote line leaves the block (Typora).
    if ((list || CRKind(p,@"quote")) && CRContent(_text.string,paragraph).length==0) {
        if (CRKind(p,@"quote") && [p[@"depth"] integerValue]>0) [self indentQuote:YES];
        else if (list && [p[@"depth"] integerValue]>0) [self indentList:YES];
        else if (![self leaveQuotedList:paragraph]) [self format:@"body"];
        return;
    }
    if (CRKind(p,@"opaque")) { [self source]; return; }
    NSRange content=CRContent(_text.string,paragraph);
    if (CRKind(p,@"code")) {
        // Enter on the empty last line leaves the block, like Typora.
        NSUInteger nextParagraph=NSMaxRange(paragraph);
        BOOL lastLine=nextParagraph>=_text.string.length || !CRKind(CRParagraphAt(self,nextParagraph),@"code");
        if (lastLine && content.length==0) { [self format:@"body"]; return; }
    }
    if (CRKind(p,@"body") && ![p[@"footnoteId"] length] && !selection.length && selection.location==NSMaxRange(content) && [self blockRuleOnEnter:paragraph]) return;
    if (CRKind(p,@"body")) {
        // ``` followed by Enter opens a code block, the way Typora does.
        NSString *text=[_text.string substringWithRange:content];
        NSTextCheckingResult *fence=[[NSRegularExpression regularExpressionWithPattern:@"^(```|~~~)([A-Za-z0-9_+.\\-]*)$" options:0 error:NULL] firstMatchInString:text options:0 range:NSMakeRange(0,text.length)];
        if (fence) {
            NSString *mark=[text substringWithRange:[fence rangeAtIndex:1]];
            NSMutableDictionary *cp=[CRInFootnote(@{@"kind":@"code",@"open":[mark stringByAppendingString:[text substringWithRange:[fence rangeAtIndex:2]]],@"close":mark,@"runs":@[]},p) mutableCopy];
            if (p[@"footnoteContinuation"]) cp[@"footnoteContinuation"]=p[@"footnoteContinuation"];
            else [cp removeObjectForKey:@"footnoteContinuation"];
            NSMutableAttributedString *after=[_text.textStorage mutableCopy];
            [after replaceCharactersInRange:content withString:@""];
            NSRange remaining=[after.string paragraphRangeForRange:NSMakeRange(paragraph.location,0)];
            if (remaining.length) [after addAttribute:CRParagraph value:cp range:remaining];
            [self replace:NSMakeRange(0,_text.string.length) with:after selection:NSMakeRange(paragraph.location,0)];
            _text.typingAttributes=[self visual:@{CRParagraph:cp} paragraph:cp];
            if (paragraph.location==_text.string.length) _emptyParagraph=cp;
            return;
        }
    }
    if (selection.length==0 && selection.location==paragraph.location && content.length && !list && !CRKind(p,@"body") && !CRKind(p,@"quote")) {
        // Enter at the start of a heading opens an empty body line
        // above it and leaves the block itself intact.
        NSDictionary *body=CRInFootnote(CRBody(),p);
        NSAttributedString *line=[[NSAttributedString alloc] initWithString:@"\n" attributes:[self visual:@{CRParagraph:body} paragraph:body]];
        _suppress=YES;
        [self replace:selection with:line selection:NSMakeRange(selection.location+1,0)];
        _text.typingAttributes=[self visual:@{CRParagraph:p} paragraph:p];
        _suppress=NO; [self restyleDirty]; [self renumber]; [self notify];
        return;
    }
    // A rule is one unit: Enter inside "---" opens a line below it rather
    // than splitting it into "--" and "-".
    if (CRKind(p,@"source") && !selection.length && selection.location>content.location && selection.location<NSMaxRange(content) && CRRuleText([_text.string substringWithRange:content])) {
        selection=NSMakeRange(NSMaxRange(content),0); _text.selectedRange=selection;
    }
    // The end of a standalone source line (a rule, an unloaded image) is
    // the end of its block: Enter continues with prose.
    BOOL sourceEnd=CRKind(p,@"source") && selection.location==NSMaxRange(content) && (CRBlockEnding(p) || CRRuleText([_text.string substringWithRange:content]));
    // Splitting a rule (over a selection) leaves text, not more source.
    BOOL ruleSource=CRKind(p,@"source") && [p[@"raw"] isKindOfClass:NSString.class] && (CRRuleText(p[@"raw"]) || CRStaleMarker(p,@""));
    NSDictionary *next=(list || CRKind(p,@"quote") || (CRKind(p,@"source") && !sourceEnd && !ruleSource) || CRKind(p,@"code")) ? CRFresh(p) : CRInFootnote(CRBody(),p);
    if (emptyFootnote) next=CRBody();
    // The block goes on in the new item or quote line: the blank line that
    // ended it moves down, or the split would end the list (or quote) here.
    if ((list || CRKind(p,@"quote")) && [p[@"separator"] length] && ![next[@"separator"] length]) {
        NSMutableDictionary *moved=[next mutableCopy]; moved[@"separator"]=p[@"separator"]; next=moved;
        NSMutableDictionary *kept=[p mutableCopy]; [kept removeObjectForKey:@"separator"]; p=kept;
        NSRange head=NSMakeRange(paragraph.location,selection.location-paragraph.location);
        if (head.length) { [_text.textStorage addAttribute:CRParagraph value:CRStoredParagraph(p) range:head]; [self markDirty:head]; }
    }
    NSDictionary *attrs=[self visual:@{CRParagraph:p} paragraph:p];
    NSAttributedString *newline=[[NSAttributedString alloc] initWithString:@"\n" attributes:attrs];
    _suppress=YES;
    [self replace:selection with:newline selection:NSMakeRange(selection.location+1,0)];
    NSRange nextRange=[_text.string paragraphRangeForRange:NSMakeRange(selection.location+1,0)];
    if (nextRange.length) [_text.textStorage addAttribute:CRParagraph value:next range:nextRange];
    _text.typingAttributes=[self visual:@{CRParagraph:next} paragraph:next]; _emptyParagraph=next;
    _suppress=NO; [self renumber]; [self restyleDirty]; [self notify];
}
// renumber keeps each run of ordered items counting up from its first
// ordinal, so inserting or removing an item never leaves a duplicate number.
- (void)renumber { [self renumberIn:NSMakeRange(0,_text.string.length)]; }
- (void)renumberIn:(NSRange)scope {
    NSMutableDictionary<NSNumber *,NSNumber *> *expected=[NSMutableDictionary dictionary];
    NSMutableDictionary<NSNumber *,NSString *> *containers=[NSMutableDictionary dictionary];
    NSDictionary *previous=nil;
    BOOL wasSuppress=_suppress; _suppress=YES;
    for (NSValue *v in CRRangesIn(_text.string,scope)) {
        NSRange r=v.rangeValue;
        NSDictionary *p=CRParagraphAt(self,r.location);
        if (CRKind(p,@"code") && [p[@"listItem"] length]) continue; // code inside an item keeps the count going
        if (!CRList(p)) { [expected removeAllObjects]; [containers removeAllObjects]; previous=nil; continue; }
        if (previous && ((p[@"listId"] && previous[@"listId"] && ![p[@"listId"] isEqual:previous[@"listId"]]) ||
            (!p[@"listId"] && !previous[@"listId"] && CRBlockEnding(previous)))) { [expected removeAllObjects]; [containers removeAllObjects]; }
        NSNumber *depth=@(MAX(0,[p[@"depth"] integerValue]));
        for (NSNumber *key in containers.allKeys) if (key.integerValue>depth.integerValue) { [expected removeObjectForKey:key]; [containers removeObjectForKey:key]; }
        NSString *container=p[@"listContainer"] ?: @"";
        if (containers[depth] && ![containers[depth] isEqual:container]) [expected removeObjectForKey:depth];
        containers[depth]=container;
        previous=p;
        if ([p[@"listContinuation"] boolValue]) continue;
        if (!CROrdered(p)) { [expected removeObjectForKey:depth]; continue; }
        NSInteger ordinal=[p[@"ordinal"] integerValue];
        NSInteger number=expected[depth] ? expected[depth].integerValue : MAX(ordinal,1);
        if (ordinal!=number) {
            NSMutableDictionary *updated=[p mutableCopy]; updated[@"ordinal"]=@(number);
            if (r.length) { [_text.textStorage addAttribute:CRParagraph value:updated range:r]; [self markDirty:r]; }
            else { NSMutableDictionary *typing=[_text.typingAttributes mutableCopy]; typing[CRParagraph]=updated; _text.typingAttributes=typing; _emptyParagraph=updated; }
        }
        expected[depth]=@(number+1);
    }
    _suppress=wasSuppress;
    [_text setNeedsDisplay:YES];
}
#include "table_editing_darwin.h"
#include "list_editing_darwin.h"
#include "code_editing_darwin.h"
#include "code_controls_darwin.h"
#include "container_editing_darwin.h"
#include "math_document_darwin.h"
#include "inline_source_darwin.h"
#include "footnote_darwin.h"
#include "alert_darwin.h"
#include "image_darwin.h"
#include "toc_darwin.h"
#include "page_card_darwin.h"
#include "block_rules_darwin.h"
#include "slash_menu_darwin.h"
#include "format_bar_darwin.h"
#include "block_actions_darwin.h"
#include "emoji_menu_darwin.h"
#include "link_menu_darwin.h"

- (BOOL)moveTableCell:(BOOL)backward {
    if (!_text.editable || _text.hasMarkedText) return NO;
    NSTextTableBlock *cell = CRCellAt(self,_text.selectedRange.location);
    if (!cell) return NO;
    NSArray *ranges = CRRanges(_text.string);
    for (NSUInteger index=0; index<ranges.count; index++) {
        NSRange range = [ranges[index] rangeValue];
        if (CRCellAt(self,range.location) != cell) continue;
        NSInteger target = (NSInteger)index + (backward ? -1 : 1);
        if (target >= 0 && target < ranges.count) {
            NSRange next = [ranges[target] rangeValue];
            if (CRCellAt(self,next.location).table == cell.table || backward) {
                // Land after the cell's text, ready to continue typing.
                _text.selectedRange = NSMakeRange(NSMaxRange(CRContent(_text.string,next)),0);
                [_text scrollRangeToVisible:_text.selectedRange];
                return YES;
            }
        }
        if (backward) { _text.selectedRange=NSMakeRange(range.location,0); return YES; }
        NSRange tableRange = CRTableRangeNear(self,cell.table,range.location);
        NSMutableDictionary *paragraph = [[self documentForRange:tableRange][@"paragraphs"][0] mutableCopy];
        NSMutableArray *rows = [paragraph[@"rows"] mutableCopy];
        NSMutableArray *row = [NSMutableArray array];
        for (NSInteger column=0; column<cell.table.numberOfColumns; column++) [row addObject:@{@"runs":@[]}];
        [rows addObject:row]; paragraph[@"rows"] = rows;
        NSDictionary *ending = _emptyParagraph;
        NSAttributedString *value = [self attributedDocument:@{@"paragraphs":@[paragraph]} replacingRange:tableRange];
        _emptyParagraph = ending;
        [self replace:tableRange with:value selection:NSMakeRange(NSMaxRange(tableRange),0)];
        [self updateTypingParagraph];
        [_text scrollRangeToVisible:_text.selectedRange];
        return YES;
    }
    return YES;
}
// Typing `\` then ASCII punctuation is a Markdown escape: keep only the
// character, marked literal, as Typora shows it outside source mode.
- (BOOL)escapeRule:(NSString *)prefix start:(NSUInteger)start {
    NSUInteger n=prefix.length;
    if (n<2 || [prefix characterAtIndex:n-2]!='\\') return NO;
    unichar c=[prefix characterAtIndex:n-1];
    if (c>127 || ![NSCharacterSet.punctuationCharacterSet characterIsMember:c] && ![NSCharacterSet.symbolCharacterSet characterIsMember:c]) return NO;
    NSUInteger slashes=0;
    while (slashes<n-1 && [prefix characterAtIndex:n-2-slashes]=='\\' && ![_text.textStorage attribute:CRLiteral atIndex:start+n-2-slashes effectiveRange:NULL]) slashes++;
    if (slashes%2==0) return NO;
    NSRange range=NSMakeRange(start+n-2,2);
    NSMutableDictionary *attrs=[[_text.textStorage attributesAtIndex:range.location+1 effectiveRange:NULL] mutableCopy];
    NSDictionary *typing=_text.typingAttributes;
    attrs[CRLiteral]=@YES;
    [self replace:range with:[[NSAttributedString alloc] initWithString:[NSString stringWithCharacters:&c length:1] attributes:attrs] selection:NSMakeRange(range.location+1,0)];
    NSMutableDictionary *next=[typing mutableCopy]; [next removeObjectForKey:CRLiteral];
    _text.typingAttributes=next;
    return YES;
}
- (void)applyInputRules {
    if (_text.hasMarkedText || !_text.editable || _text.selectedRange.length) return;
    NSUInteger caret = _text.selectedRange.location;
    if (_text.typingAttributes[CRInlineSource] || (caret && [_text.textStorage attribute:CRInlineSource atIndex:caret-1 effectiveRange:NULL])) return;
    NSRange paragraphRange = [_text.string paragraphRangeForRange:NSMakeRange(caret,0)];
    NSDictionary *paragraph = CRParagraphAt(self,paragraphRange.location);
    if (CRKind(paragraph,@"opaque") || CRKind(paragraph,@"source") || CRKind(paragraph,@"code")) return;
    NSRange beforeCaret = NSMakeRange(paragraphRange.location,caret-paragraphRange.location);
    NSString *prefix = [_text.string substringWithRange:beforeCaret];
    if (_text.typingAttributes[CRCode] || (caret && [[_text.textStorage attribute:CRCode atIndex:caret-1 effectiveRange:NULL] boolValue])) return;
    if ([self emojiRule:YES]) return;
    if ([self escapeRule:prefix start:paragraphRange.location]) return;
    // A typed `<br>` is the HTML spelling of Shift+Return.
    NSTextCheckingResult *br=[[NSRegularExpression regularExpressionWithPattern:@"<br\\s*/?>$" options:NSRegularExpressionCaseInsensitive error:NULL] firstMatchInString:prefix options:0 range:NSMakeRange(0,prefix.length)];
    if (br && !CRCellAt(self,caret-1) && ![_text.textStorage attribute:CRLiteral atIndex:paragraphRange.location+br.range.location effectiveRange:NULL]
        && !(br.range.location && [prefix characterAtIndex:br.range.location-1]=='\\')) {
        unichar ls=0x2028;
        NSMutableDictionary *attrs=[_text.typingAttributes mutableCopy]; [attrs removeObjectForKey:CRLiteral];
        NSRange range=NSMakeRange(paragraphRange.location+br.range.location,br.range.length);
        [self replace:range with:[[NSAttributedString alloc] initWithString:[NSString stringWithCharacters:&ls length:1] attributes:attrs] selection:NSMakeRange(range.location+1,0)];
        _text.typingAttributes=attrs; return;
    }
    if ([self autolinkRule:prefix start:paragraphRange.location]) return;
    // `[^1]` typed at line start already became a reference; `: ` after it
    // still means a definition.
    if (prefix.length==3 && [prefix hasSuffix:@": "]) {
        NSString *label=[_text.textStorage attribute:CRFootnote atIndex:paragraphRange.location effectiveRange:NULL][@"footnote"];
        if (label.length) prefix=[NSString stringWithFormat:@"[^%@]: ",label];
    }
    if (CRKind(paragraph,@"body") && ![paragraph[@"footnoteId"] length] && [prefix hasPrefix:@"[^"] && [prefix hasSuffix:@"]: "]) {
        // An empty definition is not a footnote to the parser; give it a body.
        char *json=cr_parse_markdown((char *)[prefix stringByAppendingString:@"x"].UTF8String);
        NSDictionary *doc=json ? [NSJSONSerialization JSONObjectWithData:[[NSString stringWithUTF8String:json] dataUsingEncoding:NSUTF8StringEncoding] options:0 error:NULL] : nil;
        if (json) free(json);
        NSArray *ps=doc[@"paragraphs"];
        if (ps.count==1 && [ps[0][@"footnoteId"] length]) {
            NSMutableDictionary *p=[ps[0] mutableCopy]; p[@"footnoteGroup"]=NSUUID.UUID.UUIDString; p[@"runs"]=@[];
            [p removeObjectForKey:@"footnoteRaw"]; [p removeObjectForKey:@"footnoteBaseline"];
            if (paragraph[@"separator"]) p[@"separator"]=paragraph[@"separator"];
            NSDictionary *attrs=[self visual:@{CRParagraph:p} paragraph:p];
            [self replace:beforeCaret with:[[NSAttributedString alloc] initWithString:@"" attributes:attrs] selection:NSMakeRange(beforeCaret.location,0)];
            NSRange remaining=[_text.string paragraphRangeForRange:NSMakeRange(beforeCaret.location,0)];
            if (remaining.length) [_text.textStorage addAttribute:CRParagraph value:p range:remaining];
            else _emptyParagraph=p;
            _text.typingAttributes=attrs; [self restyleDirty]; return;
        }
    }
    NSString *kind = nil;
    NSInteger level=0, ordinal=0;
    BOOL checked=NO;
    // `\#`, `\-`, `\>` typed at line start keep the marker as text.
    BOOL literalStart=prefix.length && [_text.textStorage attribute:CRLiteral atIndex:paragraphRange.location effectiveRange:NULL];
    if (literalStart) {}
    else if (CRKind(paragraph,@"body")) {
        NSTextCheckingResult *heading = [[NSRegularExpression regularExpressionWithPattern:@"^(#{1,6}) $" options:0 error:NULL] firstMatchInString:prefix options:0 range:NSMakeRange(0,prefix.length)];
        NSTextCheckingResult *ordered = [[NSRegularExpression regularExpressionWithPattern:@"^([0-9]{1,9})[.)] $" options:0 error:NULL] firstMatchInString:prefix options:0 range:NSMakeRange(0,prefix.length)];
        if (heading) { kind=@"heading"; level=[heading rangeAtIndex:1].length; }
        else if (ordered) { kind=@"ordered"; ordinal=[[prefix substringWithRange:[ordered rangeAtIndex:1]] integerValue]; }
        else if ([@[@"- ",@"* ",@"+ "] containsObject:prefix]) kind=@"bullet";
        else if ([prefix isEqual:@"> "]) kind=@"quote";
    }
    if (!literalStart && CRKind(paragraph,@"quote") && [prefix isEqual:@"> "]) kind=@"quote";
    NSInteger quoteLevel=0;
    if (!literalStart && !kind && CRKind(paragraph,@"quote") && ![paragraph[@"alert"] length] && ![paragraph[@"codeIndent"] length] && [prefix hasSuffix:@" "]) {
        // "- ", "1. ", "[ ] " inside a quote start a list in that quote (not
        // in a quote inside a list item, which cannot hold one here).
        NSTextCheckingResult *ordered = [[NSRegularExpression regularExpressionWithPattern:@"^([0-9]{1,9})[.)] $" options:0 error:NULL] firstMatchInString:prefix options:0 range:NSMakeRange(0,prefix.length)];
        if ([@[@"- ",@"* ",@"+ "] containsObject:prefix]) kind=@"bullet";
        else if (ordered) { kind=@"ordered"; ordinal=[[prefix substringWithRange:[ordered rangeAtIndex:1]] integerValue]; }
        if (kind) quoteLevel=[paragraph[@"depth"] integerValue]+1;
    }
    if (!literalStart && (CRKind(paragraph,@"body") || CRKind(paragraph,@"bullet") || CRKind(paragraph,@"ordered")) && ([@[@"[ ] ",@"[x] ",@"[X] "] containsObject:prefix])) {
        kind=@"checklist"; checked=![[prefix substringWithRange:NSMakeRange(1,1)] isEqual:@" "];
        if (CROrdered(paragraph)) ordinal=[paragraph[@"ordinal"] integerValue];
    }
    if (kind) {
        NSMutableDictionary *updated = [paragraph mutableCopy]; updated[@"kind"]=kind;
        if (quoteLevel) { updated[@"quote"]=@(quoteLevel); [updated removeObjectForKey:@"depth"]; [updated removeObjectForKey:@"raw"]; }
        if ([kind isEqual:@"quote"] && CRKind(paragraph,@"quote")) updated[@"depth"]=@([paragraph[@"depth"] integerValue]+1);
        if ([@[@"bullet",@"ordered",@"checklist"] containsObject:kind]) {
            if (!updated[@"listItem"]) updated[@"listItem"]=NSUUID.UUID.UUIDString;
            if (!updated[@"listId"]) updated[@"listId"]=NSUUID.UUID.UUIDString;
            if (!updated[@"listContainer"]) updated[@"listContainer"]=updated[@"listId"];
        }
        [updated removeObjectForKey:@"level"]; [updated removeObjectForKey:@"ordinal"]; [updated removeObjectForKey:@"checked"];
        if (level) updated[@"level"]=@(level);
        // A heading leaves the list item or quote the line sat in.
        if ([kind isEqual:@"heading"]) for (NSString *field in @[@"quote",@"codeIndent",@"itemDepth",@"depth",@"listId",@"listIndent",@"listBaseline",@"listContainer",@"listMarker",@"listItem",@"listContinuation"]) [updated removeObjectForKey:field];
        if ([kind isEqual:@"ordered"] || ([kind isEqual:@"checklist"] && ordinal>0)) updated[@"ordinal"]=@(ordinal);
        // Keep the delimiter the writer chose: "* ", "+ ", "1) ".
        if ([kind isEqual:@"bullet"]) updated[@"listMarker"]=[prefix substringToIndex:1];
        if ([kind isEqual:@"ordered"]) updated[@"listMarker"]=[prefix substringWithRange:NSMakeRange(prefix.length-2,1)];
        if ([kind isEqual:@"checklist"]) updated[@"checked"]=@(checked);
        NSMutableAttributedString *after = [_text.textStorage mutableCopy];
        [after replaceCharactersInRange:beforeCaret withString:@""];
        NSRange remaining = [after.string paragraphRangeForRange:NSMakeRange(paragraphRange.location,0)];
        if (remaining.length) [after addAttribute:CRParagraph value:updated range:remaining];
        [self replace:NSMakeRange(0,_text.string.length) with:after selection:NSMakeRange(paragraphRange.location,0)];
        _text.typingAttributes = [self visual:@{CRParagraph:updated} paragraph:updated];
        if (paragraphRange.location == _text.string.length) _emptyParagraph=updated;
        return;
    }
    // A paragraph that is exactly "$$" closes a math block: find its opener
    // above and render the whole block at once.
    NSString *line=[_text.string substringWithRange:CRContent(_text.string,paragraphRange)];
    if ([line isEqual:@"$$"] && (CRKind(paragraph,@"body") || CRKind(paragraph,@"source"))) {
        NSArray *ranges=CRRanges(_text.string);
        NSInteger here=-1,open=-1;
        for (NSInteger i=0;i<ranges.count;i++) {
            NSRange r=[ranges[i] rangeValue];
            if (r.location==paragraphRange.location) { here=i; break; }
        }
        for (NSInteger i=here-1;i>=0 && open<0;i--) {
            NSRange r=CRContent(_text.string,[ranges[i] rangeValue]);
            NSDictionary *p=CRParagraphAt(self,r.location);
            if (!CRKind(p,@"body") && !CRKind(p,@"source")) break;
            NSString *s=[_text.string substringWithRange:r];
            if ([s isEqual:@"$$"]) open=i;
            if (CRBlockEnding(p)) break;
        }
        if (open>=0) {
            NSRange whole=NSMakeRange([ranges[open] rangeValue].location,NSMaxRange(paragraphRange)-[ranges[open] rangeValue].location);
            if ([self renderSource:[_text.string substringWithRange:CRContent(_text.string,whole)] inRange:whole]) return;
        }
    }
    // A line that is exactly ![alt](src) becomes the image figure.
    if (CRKind(paragraph,@"body") && ![paragraph[@"footnoteId"] length] && caret==NSMaxRange(CRContent(_text.string,paragraphRange)) && [prefix hasSuffix:@")"]
        && [[NSRegularExpression regularExpressionWithPattern:@"^!\\[[^\\]\\n]*\\]\\([^)\\n]+\\)$" options:0 error:NULL] firstMatchInString:prefix options:0 range:NSMakeRange(0,prefix.length)]
        ) {
        if ([self renderSource:prefix inRange:paragraphRange]) return;
        // An image that cannot load stays verbatim, as the loader shows it.
        NSAttributedString *value=[self sourceText:@{@"kind":@"source",@"raw":prefix,@"separator":@"\n\n"}];
        [self replace:beforeCaret with:value selection:NSMakeRange(beforeCaret.location+value.length,0)];
        [self updateTypingParagraph]; return;
    }
    NSArray *patterns = @[@"\\[\\^[^\\] \\t\\n]+\\]$",@"\\$[^$\\n]+\\$$", @"\\*\\*[^*\\n]+\\*\\*$", @"__[^_\\n]+__$", @"~~[^~\\n]+~~$", @"`[^`\\n]+`$", @"\\*[^*\\n]+\\*$", @"_[^_\\n]+_$", @"==[^=\\n]+==$", @"\\^[^^\\n]+\\^$", @"~[^~\\n]+~$", @"<u>[^\\n]+</u>$", @"<kbd>[^\\n]+</kbd>$", @"<mark>[^\\n]+</mark>$", @"<sup>[^\\n]+</sup>$", @"<sub>[^\\n]+</sub>$", @"\\[[^\\]\\n]+\\]\\([^\\n]+\\)$"];
    for (NSString *pattern in patterns) {
        NSTextCheckingResult *match = [[NSRegularExpression regularExpressionWithPattern:pattern options:0 error:NULL] firstMatchInString:prefix options:0 range:NSMakeRange(0,prefix.length)];
        if (!match) continue;
        NSUInteger start=match.range.location, slashCount=0;
        while (start > slashCount && [prefix characterAtIndex:start-slashCount-1]=='\\' && ![_text.textStorage attribute:CRLiteral atIndex:beforeCaret.location+start-slashCount-1 effectiveRange:NULL]) slashCount++;
        if (slashCount % 2 || (start && [prefix characterAtIndex:start-1]==[prefix characterAtIndex:start])) continue;
        if ([_text.textStorage attribute:CRLiteral atIndex:beforeCaret.location+start effectiveRange:NULL] || [_text.textStorage attribute:CRLiteral atIndex:beforeCaret.location+NSMaxRange(match.range)-1 effectiveRange:NULL]) continue;
        if (start && [prefix characterAtIndex:start]=='_' && [NSCharacterSet.alphanumericCharacterSet characterIsMember:[prefix characterAtIndex:start-1]]) continue;
        if (start && [prefix characterAtIndex:start]=='[' && [prefix characterAtIndex:start-1]=='!') continue; // ![alt](src) is an image, not a link
        NSString *source = [prefix substringWithRange:match.range];
        NSTextTableBlock *cell=CRCellAt(self,beforeCaret.location+match.range.location);
        char *json = cr_parse_inline((char *)source.UTF8String,cell?1:0);
        if (!json) continue;
        NSDictionary *doc = [NSJSONSerialization JSONObjectWithData:[[NSString stringWithUTF8String:json] dataUsingEncoding:NSUTF8StringEncoding] options:0 error:NULL]; free(json);
        NSArray *paragraphs = doc[@"paragraphs"];
        if (paragraphs.count != 1 || !CRKind(paragraphs[0],@"body")) continue;
        NSArray *runs = paragraphs[0][@"runs"];
        BOOL styled=NO;
        for (NSDictionary *run in runs) styled |= [run[@"bold"] boolValue] || [run[@"italic"] boolValue] || [run[@"strike"] boolValue] || [run[@"code"] boolValue] || [run[@"underline"] boolValue] || [run[@"kbd"] boolValue] || [run[@"highlight"] boolValue] || [run[@"sup"] boolValue] || [run[@"sub"] boolValue] || [run[@"link"] length] > 0 || [run[@"math"] length] > 0 || [run[@"footnote"] length] > 0;
        if (!styled) continue;
        NSRange range=NSMakeRange(beforeCaret.location+match.range.location,match.range.length);
        NSDictionary *extra=cell ? @{CRTableCell:cell} : @{};
        NSArray *previousMath=_preparedMath; NSUInteger previousIndex=_mathIndex;
        _preparedMath=[self mathForFragment:doc replacingRange:range]; _mathIndex=0;
        NSAttributedString *value=[self attributedRuns:runs paragraph:paragraph extra:extra];
        _preparedMath=previousMath; _mathIndex=previousIndex;
        NSMutableDictionary *typing=[_text.typingAttributes mutableCopy];
        for (NSString *key in @[CRInlineSource,CRMathText,CRMathExpr,CRMathNumber,CRMathRendered,NSAttachmentAttributeName]) [typing removeObjectForKey:key];
        [self replace:range with:value selection:NSMakeRange(range.location+value.length,0)];
        _text.typingAttributes=typing;
        return;
    }
}
- (NSRect)markerLineRect:(NSRange)range {
    // The first line's box, from its baseline and font: glyph bounds and
    // fragment rects carry lineSpacing and newline extents that depend on
    // whether more text follows, so markers jumped as a list grew.
    NSLayoutManager *layout=_text.layoutManager;
    NSRange glyph=[layout glyphRangeForCharacterRange:CRContent(_text.string,range) actualCharacterRange:NULL];
    NSFont *font;
    NSRect line;
    CGFloat baseline;
    if (!glyph.length && !range.length) {
        // The trailing empty paragraph is the extra line fragment, which only
        // exists once layout reaches the end.
        [layout ensureLayoutForCharacterRange:NSMakeRange(_text.string.length,0)];
        font=_text.typingAttributes[NSFontAttributeName] ?: _text.font;
        line=layout.extraLineFragmentUsedRect;
        baseline=font.ascender;
    } else {
        NSUInteger index=glyph.length ? glyph.location : [layout glyphIndexForCharacterAtIndex:range.location];
        font=[_text.textStorage attribute:NSFontAttributeName atIndex:[layout characterIndexForGlyphAtIndex:index] effectiveRange:NULL] ?: _text.font;
        NSRange lineGlyphs;
        line=[layout lineFragmentUsedRectForGlyphAtIndex:index effectiveRange:&lineGlyphs];
        baseline=[layout locationForGlyphAtIndex:index].y;
        // The used rect starts at the indented first glyph; markers sit left of it.
        if (glyph.length) line.origin.x=[layout boundingRectForGlyphRange:NSIntersectionRange(lineGlyphs,glyph) inTextContainer:_text.textContainer].origin.x;
    }
    NSRect box=NSMakeRect(line.origin.x,line.origin.y+baseline-font.ascender,line.size.width,font.ascender-font.descender);
    return NSOffsetRect(box,_text.textContainerOrigin.x,_text.textContainerOrigin.y);
}
- (NSRect)markerRect:(NSRange)range {
    NSRect line=[self markerLineRect:range];
    NSDictionary *p=CRParagraphAt(self,range.location);
    CGFloat indent=CRList(p) ? (MAX(0,[p[@"depth"] integerValue])+CRQuoteOf(p))*26*_zoom : CRCodeListIndent(p)*_zoom; // a quote in an item draws its bars at the item text
    if (CRKind(p,@"checklist") && CROrdered(p)) indent+=24*_zoom;
    return NSMakeRect(_text.textContainerOrigin.x+4*_zoom+indent,NSMidY(line)-9.5*_zoom,19*_zoom,19*_zoom);
}
- (void)toggleCheckbox:(NSUInteger)location {
    if (!_text.editable || _text.hasMarkedText) return;
    NSRange range=[_text.string paragraphRangeForRange:NSMakeRange(location,0)];
    NSMutableDictionary *p=[CRParagraphAt(self,location) mutableCopy];
    if (!CRKind(p,@"checklist") || [p[@"listContinuation"] boolValue]) return;
    p[@"checked"]=[p[@"checked"] boolValue] ? @NO : @YES;
    NSMutableAttributedString *after=[_text.textStorage mutableCopy];
    [after addAttribute:CRParagraph value:p range:range];
    [self replace:NSMakeRange(0,_text.string.length) with:after selection:_text.selectedRange];
}

// sourceText renders a block's raw source as editable source lines. It backs
// the figure fallback (a formula that will not render, a missing image) and
// the click-to-reveal editing of rendered formulas.
- (NSAttributedString *)sourceText:(NSDictionary *)p {
    NSString *raw=[p[@"raw"] isKindOfClass:NSString.class] ? p[@"raw"] : @"";
    NSArray *lines=[raw componentsSeparatedByString:@"\n"];
    NSMutableAttributedString *value=[NSMutableAttributedString new];
    for (NSUInteger i=0;i<lines.count;i++) {
        NSString *text=lines[i];
        if ([text hasSuffix:@"\r"]) text=[text substringToIndex:text.length-1];
        NSString *sep=i+1<lines.count ? @"\n" : ([p[@"separator"] isKindOfClass:NSString.class] ? p[@"separator"] : @"");
        NSDictionary *lp=CRInFootnote(@{@"kind":@"source",@"runs":@[@{@"text":text}],@"raw":text,@"separator":sep},p);
        [value appendAttributedString:[[NSAttributedString alloc] initWithString:text attributes:[self visual:@{CRParagraph:lp} paragraph:lp]]];
        if (i+1<lines.count) [value appendAttributedString:[[NSAttributedString alloc] initWithString:@"\n" attributes:[self visual:@{CRParagraph:lp} paragraph:lp]]];
    }
    if ([p[@"footnoteId"] length] && value.length) [value addAttribute:CRFootnoteFigureSource value:@{@"id":NSUUID.UUID.UUIDString,@"paragraph":p,@"source":value.string} range:NSMakeRange(0,value.length)];
    return value;
}
- (void)revealSourceAt:(NSUInteger)location {
    if (!_text.editable || _text.hasMarkedText) return;
    NSRange paragraph=[_text.string paragraphRangeForRange:NSMakeRange(location,0)];
    NSAttributedString *value=[self sourceText:CRParagraphAt(self,paragraph.location)];
    if (!value.length) return;
    [self replace:CRContent(_text.string,paragraph) with:value selection:NSMakeRange(paragraph.location,0)];
    // The replace ran suppressed, so the selection change never armed a draft
    // group. Arm it now so leaving the revealed source re-renders the figure.
    _draftGroup=[self draftGroupAt:_text.selectedRange.location];
}

- (NSRect)imageBounds:(NSImage *)img spec:(NSString *)spec {
    return CRImageBoundsFromSpec(img, spec, MAX(60,_text.textContainer.size.width-8*_zoom));
}
- (NSImage *)placeholderImage {
    return [NSImage imageWithSize:NSMakeSize(240*_zoom,120*_zoom) flipped:NO drawingHandler:^BOOL(NSRect r) {
        [[NSColor colorWithWhite:_dark ? 0.23 : 0.94 alpha:1] setFill];
        [[NSBezierPath bezierPathWithRoundedRect:r xRadius:6*_zoom yRadius:6*_zoom] fill];
        return YES;
    }];
}

// draftGroupAt finds the contiguous run of source/body paragraphs around
// location: the region a Typora-style re-render may safely reconsider.
- (NSRange)draftGroupAt:(NSUInteger)location {
    NSArray *ranges=CRRanges(_text.string);
    NSInteger index=-1;
    for (NSInteger i=0;i<ranges.count;i++) {
        NSRange r=[ranges[i] rangeValue];
        if (location>=r.location && (location<NSMaxRange(r) || i+1==ranges.count)) { index=i; break; }
        if (r.location>location) break;
    }
    if (index<0) return NSMakeRange(NSNotFound,0);
    // Source lines and typed body lines are separate drafts: a revealed
    // formula must not absorb the prose typed next to it.
    NSDictionary *anchor=CRParagraphAt(self,[ranges[index] rangeValue].location);
    BOOL (^drafty)(NSInteger)=^BOOL(NSInteger i){
        NSDictionary *p=CRParagraphAt(self,[ranges[i] rangeValue].location);
        NSString *group=anchor[@"footnoteGroup"];
        BOOL sameGroup=(!group && !p[@"footnoteGroup"]) || [group isEqual:p[@"footnoteGroup"]];
        return sameGroup && (CRKind(p,@"source") || CRKind(p,@"body")) && [p[@"kind"] isEqual:anchor[@"kind"]];
    };
    if (!drafty(index)) return NSMakeRange(NSNotFound,0);
    NSInteger start=index,end=index;
    // Block separators (blank lines) bound the group.
    while (start>0 && drafty(start-1) && !CRBlockEnding(CRParagraphAt(self,[ranges[start-1] rangeValue].location))) start--;
    while (end+1<ranges.count && drafty(end+1) && !CRBlockEnding(CRParagraphAt(self,[ranges[end] rangeValue].location))) end++;
    return NSMakeRange([ranges[start] rangeValue].location,NSMaxRange([ranges[end] rangeValue])-[ranges[start] rangeValue].location);
}

// renderSource re-parses text; if it is exactly renderable blocks (math,
// image, fenced code), range becomes its rendered form again and returns YES.
// Anything else stays editable text.
- (BOOL)renderSource:(NSString *)text inRange:(NSRange)range {
    if (!text.length) return NO;
    NSDictionary *origin=range.location<_text.string.length ? [_text.textStorage attribute:CRFootnoteFigureSource atIndex:range.location effectiveRange:NULL][@"paragraph"] : nil;
    if ([origin[@"raw"] containsString:@"\r\n"]) text=[text stringByReplacingOccurrencesOfString:@"\n" withString:@"\r\n"];
    char *json=cr_parse_markdown((char *)text.UTF8String);
    if (!json) return NO;
    NSMutableDictionary *doc=[[NSJSONSerialization JSONObjectWithData:[[NSString stringWithUTF8String:json] dataUsingEncoding:NSUTF8StringEncoding] options:0 error:NULL] mutableCopy];
    free(json);
    NSArray *ps=doc[@"paragraphs"];
    if (!ps.count) return NO;
    BOOL renderable=YES;
    NSMutableArray *paragraphs=[NSMutableArray array];
    for (NSDictionary *p in ps) {
        if (!CRKind(p,@"math") && !CRKind(p,@"image") && !CRKind(p,@"code") && !CRKind(p,@"diagram") && !CRKind(p,@"toc") && !CRKind(p,@"page")) renderable=NO;
        // Fresh parse IDs collide with the document's; identity stays valid
        // through baseline, not id.
        NSMutableDictionary *clean=[p mutableCopy];
        [clean removeObjectForKey:@"id"]; [clean removeObjectForKey:@"nextId"];
        [paragraphs addObject:clean];
    }
    if (!renderable) return NO;
    if (origin[@"separator"] && paragraphs.count) {
        NSMutableDictionary *last=[paragraphs.lastObject mutableCopy]; last[@"separator"]=origin[@"separator"]; paragraphs[paragraphs.count-1]=last;
    }
    // Revealing a figure and folding it back is not an edit. The rebuilt block
    // is the very paragraph that was revealed, so keep its identity: without
    // id/nextId the serializer sees a changed adjacency and widens the
    // neighbouring separators into blank lines the source never had.
    if (origin && paragraphs.count==1) {
        NSMutableDictionary *only=[paragraphs.firstObject mutableCopy];
        for (NSString *key in @[@"id",@"nextId"]) if ([origin[key] isKindOfClass:NSString.class] && [origin[key] length]) only[key]=origin[key];
        paragraphs[0]=only;
    }
    doc[@"paragraphs"]=paragraphs;
    NSDictionary *ending=_emptyParagraph;
    NSAttributedString *value=[self attributedDocument:doc replacingRange:range];
    _emptyParagraph=ending;
    // A single figure that falls back to source text changes nothing.
    if (ps.count==1 && ![value attribute:NSAttachmentAttributeName atIndex:0 effectiveRange:NULL]) return NO;
    NSRange content=CRContent(_text.string,range);
    [self replace:content with:value selection:NSMakeRange(content.location+value.length,0)];
    return YES;
}

// renderDraftGroup re-renders the group the caret just left, if it parses as
// renderable blocks.
- (void)renderDraftGroup:(NSRange)group {
    // Source edits can change its length before a selection notification.
    // Derive the current block range instead of using a stale UTF-16 length.
    if (group.location==NSNotFound || group.location>_text.string.length) return;
    group=[self draftGroupAt:group.location];
    if (group.location==NSNotFound) return;
    NSString *source=[_text.string substringWithRange:CRContent(_text.string,group)];
    // An unclosed "$$" parses as a block running to the end of the fragment:
    // it stays a draft until its closing "$$" is typed.
    NSString *trimmed=[source stringByTrimmingCharactersInSet:NSCharacterSet.whitespaceAndNewlineCharacterSet];
    if ([trimmed hasPrefix:@"$$"] && (trimmed.length<4 || ![trimmed hasSuffix:@"$$"])) return;
    [self renderSource:source inRange:group];
}

// finishDraftGroup re-renders the block the caret just left, if any.
- (void)finishDraftGroup {
    if (_suppress || _text.hasMarkedText) return;
    NSRange group=_draftGroup;
    if (group.location!=NSNotFound && group.location<=_text.string.length) group=[self draftGroupAt:group.location];
    _draftGroup=NSMakeRange(NSNotFound,0);
    NSUInteger caret=_text.selectedRange.location;
    if (group.location!=NSNotFound && (caret<group.location || caret>NSMaxRange(group))) [self renderDraftGroup:group];
    _draftGroup=[self draftGroupAt:_text.selectedRange.location];
}

// the text baseline; nil falls back to the literal source text.
- (NSAttributedString *)inlineMath:(NSDictionary *)run paragraph:(NSDictionary *)paragraph extra:(NSDictionary *)extra {
    NSString *expr=run[@"math"];
    NSDictionary *prepared=[self nextMath];
    NSImage *img=[self mathImage:prepared];
    if (!img) {
        NSMutableDictionary *attrs=[@{CRParagraph:paragraph,CRInlineSource:NSUUID.UUID.UUIDString} mutableCopy];
        [attrs addEntriesFromDictionary:extra];
        if ([prepared[@"error"] length]) attrs[NSToolTipAttributeName]=prepared[@"error"];
        return [[NSAttributedString alloc] initWithString:run[@"text"] ?: @"" attributes:[self visual:attrs paragraph:paragraph]];
    }
    NSTextAttachment *att=[NSTextAttachment new];
    att.image=img;
    // Approximate baseline alignment: math images hang a fifth below it.
    att.bounds=NSMakeRect(0,-img.size.height*0.22,img.size.width,img.size.height);
    NSMutableAttributedString *s=[[NSAttributedString attributedStringWithAttachment:att] mutableCopy];
    NSMutableDictionary *attrs=[@{CRParagraph:paragraph} mutableCopy];
    [attrs addEntriesFromDictionary:extra];
    attrs[CRMathText]=run[@"text"];
    attrs[CRMathExpr]=expr;
    if (prepared) { attrs[CRMathRendered]=prepared; attrs[CRMathNumber]=prepared[@"number"] ?: @""; }
    [s addAttributes:[self visual:attrs paragraph:paragraph] range:NSMakeRange(0,s.length)];
    // visual() rebuilds the attribute dictionary; keep the identity keys.
    [s addAttributes:@{CRMathText:run[@"text"],CRMathExpr:expr} range:NSMakeRange(0,s.length)];
    return s;
}

// figure renders a math or image paragraph as an attachment; anything that
// cannot render becomes its own editable source text instead.
// imageAttachmentForURL loads a local or remote image into an attachment
// (remote ones show a placeholder until they arrive), or nil if it cannot.
- (NSTextAttachment *)imageAttachmentForURL:(NSString *)url spec:(NSString *)imageSpec {
    if (!url.length) return nil;
    NSImage *img=nil;
    id cached=_imageCache[url];
    if ([cached isKindOfClass:NSImage.class]) img=cached;
    else if (cached) return nil;
    else if ([url hasPrefix:@"http://"] || [url hasPrefix:@"https://"]) {
        img=[self placeholderImage];
        if (!_imagePending[url]) {
            _imagePending[url]=@YES;
            [[[NSURLSession sharedSession] dataTaskWithURL:[NSURL URLWithString:url] completionHandler:^(NSData *data,NSURLResponse *response,NSError *error) {
                NSImage *loaded=data.length ? [[NSImage alloc] initWithData:data] : nil;
                dispatch_async(dispatch_get_main_queue(),^{
                    self.imageCache[url]=loaded ?: (id)[NSNull null];
                    [self.imagePending removeObjectForKey:url];
                    for (NSTextAttachment *att in [self.imageAttachments[url] objectEnumerator]) if (loaded) { att.image=loaded; att.bounds=[self imageBounds:loaded spec:self.imageSpecForURL[url]]; }
                    [self.text.layoutManager invalidateLayoutForCharacterRange:NSMakeRange(0,self.text.textStorage.length) actualCharacterRange:NULL];
                    [self.text setNeedsDisplay:YES];
                });
            }] resume];
        }
        NSTextAttachment *att=[NSTextAttachment new];
        att.image=img; att.bounds=[self imageBounds:img spec:imageSpec];
        NSMutableArray *list=_imageAttachments[url];
        if (!list) { list=[NSMutableArray new]; _imageAttachments[url]=list; }
        [list addObject:att];
        self.imageSpecForURL[url]=imageSpec ?: @"";
        return att;
    } else {
        NSString *path=url;
        if (![path hasPrefix:@"/"] && _baseDir.length) path=[_baseDir stringByAppendingPathComponent:path];
        img=[[NSImage alloc] initWithContentsOfFile:path];
        if (img) _imageCache[url]=img;
        else { _imageCache[url]=[NSNull null]; return nil; }
    }
    NSTextAttachment *att=[NSTextAttachment new];
    att.image=img; att.bounds=[self imageBounds:img spec:imageSpec];
    return att;
}
- (NSAttributedString *)figure:(NSDictionary *)p {
    NSString *raw=[p[@"raw"] isKindOfClass:NSString.class] ? p[@"raw"] : @"";
    NSImage *img=nil;
    BOOL failed=NO;
    NSTextAttachment *pending=nil;
    NSString *imageSpec=nil;
    if (CRKind(p,@"diagram")) {
        NSString *key=[NSString stringWithFormat:@"d|%d|%f|%@",_dark,_zoom,raw];
        id cached=_mathCache[key];
        if ([cached isKindOfClass:NSImage.class]) img=cached;
        else if (cached) failed=YES;
        else {
            // raw is a fenced block: ```lang\n...body...\n```
            NSString *lang=@"",*body=raw;
            NSRange nl=[raw rangeOfString:@"\n"];
            if (nl.location!=NSNotFound) {
                NSString *fence=[raw substringToIndex:nl.location];
                lang=[[fence stringByTrimmingCharactersInSet:[NSCharacterSet characterSetWithCharactersInString:@"`~ "]] lowercaseString];
                body=[raw substringFromIndex:NSMaxRange(nl)];
                NSRange last=[body rangeOfString:@"\n" options:NSBackwardsSearch];
                if (last.location!=NSNotFound) body=[body substringToIndex:last.location];
            }
            char *b64=cr_render_block((char *)lang.UTF8String,(char *)body.UTF8String,_dark?1:0,_zoom);
            NSData *data=b64 ? [[NSData alloc] initWithBase64EncodedString:[NSString stringWithUTF8String:b64] options:0] : nil;
            if (b64) free(b64);
            img=data.length ? [[NSImage alloc] initWithData:data] : nil;
            if (img) { img.size=NSMakeSize(img.size.width/2,img.size.height/2); _mathCache[key]=img; } // rendered at 2x
            else { _mathCache[key]=[NSNull null]; failed=YES; }
        }
    } else if (CRKind(p,@"math")) {
        NSDictionary *prepared=[self nextMath];
        img=[self mathImage:prepared]; failed=!img;
    } else {
        // The destination may carry a title and a size suffix; only the URL
        // loads, and the suffix decides how large the figure is drawn.
        imageSpec=CRImageRawSpec(raw);
        NSString *url=[CRImageRawURL(raw) stringByRemovingPercentEncoding];
        pending=[self imageAttachmentForURL:url spec:imageSpec];
        if (!pending) failed=YES;
        else img=pending.image;
    }
    if (failed) return [self sourceText:p];
    NSTextAttachment *att=pending;
    if (!att) {
        att=[NSTextAttachment new];
        att.image=img;
        att.bounds=[self imageBounds:img spec:imageSpec];
    }
    if (!att.image) return [self sourceText:p];
    NSMutableAttributedString *s=[[NSAttributedString attributedStringWithAttachment:att] mutableCopy];
    NSMutableDictionary *attrs=[[self visual:@{CRParagraph:p} paragraph:p] mutableCopy];
    if (CRKind(p,@"math") && _preparedMath && _mathIndex>0 && _mathIndex<=_preparedMath.count) {
        NSDictionary *prepared=_preparedMath[_mathIndex-1];
        attrs[CRMathRendered]=prepared; attrs[CRMathNumber]=prepared[@"number"] ?: @"";
    }
    if (CRKind(p,@"math")) [attrs[NSParagraphStyleAttributeName] setAlignment:NSTextAlignmentCenter]; // display math centers
    [s addAttributes:attrs range:NSMakeRange(0,s.length)];
    return s;
}
@end

@implementation CRTextView
- (BOOL)becomeFirstResponder {
    NSResponder *previous=self.window.firstResponder;
    if (previous && previous!=self) self.owner.previousResponder=previous;
    return [super becomeFirstResponder];
}
- (void)insertText:(id)value replacementRange:(NSRange)range {
    // NSNotFound is the IME commit contract: super resolves it to the marked
    // range. Substituting the caret here would strand the pinyin as plain
    // text and append the composed characters after it.
    NSRange effective = range;
    if (effective.location==NSNotFound && self.hasMarkedText) effective = self.markedRange;
    if (effective.location==NSNotFound) effective = self.selectedRange;
    if ([self.owner prepareInlineReplacement:&effective]) range=effective;
    NSString *codeText=[value isKindOfClass:NSAttributedString.class] ? [value string] : value;
    if (!self.hasMarkedText && !self.owner.suppress && effective.length && [self.owner replaceAcrossTable:effective withText:codeText]) return;
    if (![self.owner allowRange:effective]) return;
    if (!self.hasMarkedText && (effective.length || [codeText rangeOfCharacterFromSet:NSCharacterSet.newlineCharacterSet].location!=NSNotFound) && [self.owner replaceCodeRange:effective withText:codeText]) return;
    BOOL opaque=NO;
    for (NSValue *v in CRRanges(self.string)) if (NSIntersectionRange(v.rangeValue,effective).length && CRKind(CRParagraphAt(self.owner,v.rangeValue.location),@"opaque")) opaque=YES;
    NSTextTableBlock *targetCell=CRCellAt(self.owner,effective.location);
    if (targetCell && effective.length) {
        NSRange tableRange=CRTableRangeNear(self.owner,targetCell.table,effective.location);
        if (effective.location <= tableRange.location && NSMaxRange(effective) >= NSMaxRange(tableRange)) opaque=YES;
    }
    if (opaque) {
        if (![self.owner allowRange:effective]) return;
        NSString *string=[value isKindOfClass:NSAttributedString.class] ? [value string] : value;
        NSDictionary *body=CRInFootnote(CRBody(),CRParagraphAt(self.owner,effective.location));
        NSDictionary *attrs=[self.owner visual:@{CRParagraph:body} paragraph:body];
        NSDictionary *ending=self.owner.emptyParagraph;
        BOOL endsAtEOF=NSMaxRange(effective)==self.string.length;
        self.owner.suppress=YES;
        [self.owner replace:effective with:[[NSAttributedString alloc] initWithString:string attributes:attrs] selection:NSMakeRange(effective.location+string.length,0)];
        self.typingAttributes=attrs; self.owner.emptyParagraph=endsAtEOF ? body : ending; self.owner.suppress=NO; [self.owner notify];
        return;
    }
    if (range.location==NSNotFound || NSEqualRanges(range,self.selectedRange)) {
        if ([self.owner openParagraphBesideFigure]) { effective=self.selectedRange; range=effective; }
    }
    if ([self.owner autoPair:value range:effective]) {
        if (!self.owner.suppress && !self.hasMarkedText) {
            [self.owner slashQueryChanged];
            [self.owner emojiMenuQueryChanged];
            [self.owner linkMenuQueryChanged];
        }
        return;
    }
    NSDictionary *destinationParagraph=CRParagraphAt(self.owner,effective.location);
    BOOL wasSuppress=self.owner.suppress;
    if (targetCell) {
        NSString *string=[value isKindOfClass:NSAttributedString.class] ? [value string] : value;
        value=[[string componentsSeparatedByCharactersInSet:NSCharacterSet.newlineCharacterSet] componentsJoinedByString:@" "];
        NSMutableDictionary *typing=[self.typingAttributes mutableCopy];
        typing[CRTableCell]=targetCell; typing[CRParagraph]=CRParagraphAt(self.owner,effective.location);
        self.typingAttributes=[self.owner visual:typing paragraph:typing[CRParagraph]];
    }
    self.owner.suppress=YES;
    [super insertText:value replacementRange:range];
    if (!wasSuppress && !self.hasMarkedText) {
        NSString *inserted=[value isKindOfClass:NSAttributedString.class] ? [value string] : value;
        if (inserted.length && [inserted rangeOfCharacterFromSet:NSCharacterSet.newlineCharacterSet].location==NSNotFound && effective.location+inserted.length<=self.string.length) {
            // AppKit can inherit the preceding item's metadata at an empty
            // paragraph. Use the destination identity after structural edits.
            [self.textStorage addAttribute:CRParagraph value:destinationParagraph range:NSMakeRange(effective.location,inserted.length)];
            // Typing over a selection that starts at a formula or footnote
            // mark takes that character's attributes: typed text is text.
            NSRange typed=NSMakeRange(effective.location,inserted.length);
            if ([self.textStorage attribute:NSAttachmentAttributeName atIndex:typed.location effectiveRange:NULL] && [inserted rangeOfString:@"\uFFFC"].location==NSNotFound) {
                [self.textStorage setAttributes:CRTextAttrs([self.textStorage attributesAtIndex:typed.location effectiveRange:NULL]) range:typed];
            }
        }
        [self.owner normalize];
        NSString *string=[value isKindOfClass:NSAttributedString.class] ? [value string] : value;
        if (string.length && [string rangeOfCharacterFromSet:NSCharacterSet.newlineCharacterSet].location==NSNotFound) [self.owner applyInputRules];
    }
    self.owner.suppress=wasSuppress;
    if (!wasSuppress) [self clearHover];
    if (!wasSuppress && !self.hasMarkedText) { [self.owner slashQueryChanged]; [self.owner emojiMenuQueryChanged]; [self.owner linkMenuQueryChanged]; }
    if (!wasSuppress) [self.owner notify];
}
- (void)insertNewline:(id)sender { [self.owner newline]; }
- (void)insertLineBreak:(id)sender {
    // Shift+Return authors a Markdown hard break: a soft visual line (U+2028)
    // inside the current block, not a new paragraph. Tables keep their own
    // row/newline semantics, so fall back there.
    if (CRCellAt(self.owner,self.selectedRange.location)) { [self.owner newline]; return; }
    NSDictionary *p=CRParagraphAt(self.owner,self.selectedRange.location);
    // A heading is one line in Markdown: Shift+Return ends it like Return.
    if (CRKind(p,@"heading") || CRKind(p,@"code") || CRKind(p,@"source") || CRKind(p,@"opaque") || CRKind(p,@"math") || CRKind(p,@"image")) { [self.owner newline]; return; }
    unichar ls=0x2028;
    [self insertText:[NSString stringWithCharacters:&ls length:1] replacementRange:self.selectedRange];
}
- (void)insertTab:(id)sender {
    if ([self.owner moveTableCell:NO]) return;
    NSDictionary *p=CRParagraphAt(self.owner,self.selectedRange.location);
    if (CRList(p)) { [self.owner indentList:NO]; return; }
    if (CRKind(p,@"code") && self.selectedRange.length && [self.owner indentCode:NO]) return;
    [super insertTab:sender];
}
- (void)insertBacktab:(id)sender {
    if ([self.owner moveTableCell:YES]) return;
    NSDictionary *p=CRParagraphAt(self.owner,self.selectedRange.location);
    if (CRList(p)) { [self.owner indentList:YES]; return; }
    if (CRKind(p,@"code") && [self.owner indentCode:YES]) return;
    [super insertBacktab:sender];
}
- (void)deleteBackward:(id)sender {
    // Between an empty auto-pair, one Backspace removes both characters.
    CREditor *e=self.owner;
    NSRange selection=self.selectedRange;
    [e prepareInlineReplacement:&selection];
    if (!e.suppress && !self.hasMarkedText) {
        NSRange edit=selection;
        if (!edit.length && edit.location && [self.string characterAtIndex:edit.location-1]=='\n') edit=NSMakeRange(edit.location-1,1);
        if (edit.length && [e replaceCodeRange:edit withText:@""]) return;
        if ([e replaceAcrossTable:selection withText:@""]) return;
    }
    if (!e.suppress && self.editable && !self.hasMarkedText && !selection.length && selection.location && [e deleteMentionBefore:selection.location]) return;
    if (!e.suppress && self.editable && !self.hasMarkedText && selection.length && selection.length==self.string.length) {
        // Clearing the whole note leaves one plain line, not the last block's kind.
        NSDictionary *body=CRBody(), *attrs=[e visual:@{CRParagraph:body} paragraph:body];
        [e replace:selection with:[[NSAttributedString alloc] initWithString:@"" attributes:attrs] selection:NSMakeRange(0,0)];
        self.typingAttributes=attrs; e.emptyParagraph=body;
        return;
    }
    if (!e.suppress && self.editable && !self.hasMarkedText && selection.length==0) {
        // Backspace at the start of a heading, list item or quote turns it
        // back into body text first, the way Typora does.
        NSRange paragraph=[self.string paragraphRangeForRange:NSMakeRange(selection.location,0)];
        NSDictionary *p=CRParagraphAt(e,paragraph.location);
        BOOL block=CRKind(p,@"heading") || CRKind(p,@"bullet") || CRKind(p,@"ordered") || CRKind(p,@"checklist") || CRKind(p,@"quote");
        if (block && selection.location==paragraph.location && !CRCellAt(e,selection.location)) {
            if (CRKind(p,@"quote") && [p[@"depth"] integerValue]>0) [e indentQuote:YES];
            else if (CRList(p) && [p[@"depth"] integerValue]>0) [e indentList:YES];
            else if (![e leaveQuotedList:paragraph]) [e format:@"body"];
            return;
        }
    }
    if (!e.suppress && [e deleteSelectedFigure]) return;
    if (!e.suppress && [e selectFigureBeside:YES]) return;
    if (!e.suppress && self.editable && !self.hasMarkedText && selection.length==0 && selection.location>0 && selection.location<self.string.length) {
        unichar open=[self.string characterAtIndex:selection.location-1], close=[self.string characterAtIndex:selection.location];
        NSString *closer=CRPairCloser([NSString stringWithCharacters:&open length:1]);
        BOOL pair=closer && [closer characterAtIndex:0]==close;
        if (pair) {
            NSDictionary *p=CRParagraphAt(e,selection.location);
            BOOL plain=!(CRKind(p,@"code") || CRKind(p,@"source") || CRKind(p,@"math") || CRKind(p,@"opaque")) && !CRCellAt(e,selection.location);
            NSDictionary *before=[self.textStorage attributesAtIndex:selection.location-1 effectiveRange:NULL];
            if (plain && !before[CRCode] && !before[CRMathText] && !before[CRInlineSource]) {
                [e replace:NSMakeRange(selection.location-1,2) with:[[NSAttributedString alloc] initWithString:@"" attributes:self.typingAttributes] selection:NSMakeRange(selection.location-1,0)];
                return;
            }
        }
    }
    BOOL source=selection.location<self.string.length && [self.textStorage attribute:CRInlineSource atIndex:selection.location effectiveRange:NULL]!=nil;
    if (!source && selection.location) source=[self.textStorage attribute:CRInlineSource atIndex:selection.location-1 effectiveRange:NULL]!=nil;
    if (source && !e.suppress && !self.hasMarkedText) {
        e.suppress=YES;
        [super deleteBackward:sender];
        e.suppress=NO;
        [e normalize]; [e notify];
    } else [super deleteBackward:sender];
    if (!e.suppress && !self.hasMarkedText) { [e slashQueryChanged]; [e emojiMenuQueryChanged]; [e linkMenuQueryChanged]; }
}
- (void)deleteForward:(id)sender {
    if (!self.owner.suppress && ([self.owner deleteSelectedFigure] || [self.owner selectFigureBeside:NO])) return;
    NSRange selection=self.selectedRange;
    [self.owner prepareInlineReplacement:&selection];
    if (!self.owner.suppress && !self.hasMarkedText) {
        NSRange edit=selection;
        if (!edit.length && edit.location<self.string.length && [self.string characterAtIndex:edit.location]=='\n') edit.length=1;
        if (edit.length && [self.owner replaceCodeRange:edit withText:@""]) return;
        if ([self.owner replaceAcrossTable:selection withText:@""]) return;
    }
    BOOL source=selection.location<self.string.length && [self.textStorage attribute:CRInlineSource atIndex:selection.location effectiveRange:NULL]!=nil;
    if (source && !self.owner.suppress && !self.hasMarkedText) {
        self.owner.suppress=YES;
        [super deleteForward:sender];
        self.owner.suppress=NO;
        [self.owner normalize]; [self.owner notify];
    } else [super deleteForward:sender];
}
- (void)setMarkedText:(id)value selectedRange:(NSRange)selection replacementRange:(NSRange)replacement {
    NSRange effective=replacement;
    if (effective.location==NSNotFound && self.hasMarkedText) effective=self.markedRange;
    if (effective.location==NSNotFound) effective=self.selectedRange;
    if (![self.owner allowRange:effective]) return;
    if (!self.hasMarkedText && [self.owner openParagraphBesideFigure]) replacement=NSMakeRange(NSNotFound,0);
    // Composition stays native inside a code body. Replacing its display
    // separator would remove a hidden fence belonging to the next block.
    if (effective.length) for (NSDictionary *block in [self.owner codeBlocks]) {
        NSRange range=[block[@"range"] rangeValue];
        if (!NSIntersectionRange(effective,range).length) continue;
        NSRange last=[[block[@"lines"] lastObject] rangeValue];
        NSUInteger end=NSMaxRange(CRContent(self.string,last));
        if (effective.location<range.location || NSMaxRange(effective)>end) return;
    }
    [super setMarkedText:value selectedRange:selection replacementRange:replacement];
    [self.owner notify];
}
- (void)unmarkText { [super unmarkText]; if (!self.owner.suppress) { [self.owner normalize]; [self.owner notify]; } }
// drawRange is the text a dirty rect can show, widened to whole paragraphs
// and back to the start of any code block or quote it cuts into, so chips
// and bars still begin where their block does.
- (NSRange)drawRange:(NSRect)dirty {
    NSLayoutManager *layout=self.layoutManager;
    NSString *string=self.string;
    NSRange glyphs=[layout glyphRangeForBoundingRect:NSOffsetRect(dirty,-self.textContainerOrigin.x,-self.textContainerOrigin.y) inTextContainer:self.textContainer];
    NSRange chars=[layout characterRangeForGlyphRange:glyphs actualGlyphRange:NULL];
    NSUInteger start=[string paragraphRangeForRange:NSMakeRange(MIN(chars.location,string.length),0)].location;
    for (int i=0;i<2 && start>0;i++) start=[string paragraphRangeForRange:NSMakeRange(start-1,0)].location;
    while (start>0) {
        NSUInteger previous=[string paragraphRangeForRange:NSMakeRange(start-1,0)].location;
        NSDictionary *p=CRParagraphAt(self.owner,previous);
        if (!CRKind(p,@"code") && !CRKind(p,@"source") && !CRKind(p,@"opaque") && !CRKind(p,@"quote")) break;
        start=previous;
    }
    NSUInteger end=MIN(NSMaxRange(chars),string.length);
    for (int i=0;i<2 && end<string.length;i++) end=NSMaxRange([string paragraphRangeForRange:NSMakeRange(end,0)]);
    return NSMakeRange(start,end-start);
}
// drawInlineChips paints inline code, <kbd> and ==highlight== as rounded
// chips hugging the text (per wrapped line), under the glyphs.
- (void)drawInlineChips:(NSRange)visible {
    CREditor *e=self.owner;
    NSLayoutManager *lm=self.layoutManager;
    NSTextStorage *storage=self.textStorage;
    if (!visible.length || NSMaxRange(visible)>storage.length) return;
    for (NSString *key in @[CRBackground,CRMention,CRHighlight,CRCode,CRKbd]) {
    BOOL tint=key==CRBackground, mention=key==CRMention, mark=key==CRHighlight || tint || mention, kbd=key==CRKbd;
    // Longest runs: kerning and other attributes must not split one chip.
    [storage enumerateAttribute:key inRange:visible options:0 usingBlock:^(id value, NSRange range, BOOL *stop) {
        NSColor *fill=tint ? CRResolvedColor(value,YES,e.dark) : nil;
        if (tint ? !fill : ![value boolValue]) return;
        if (mention && ![storage attribute:NSLinkAttributeName atIndex:range.location effectiveRange:NULL]) return;
        if (mention) fill=e.dark ? [NSColor colorWithWhite:1 alpha:0.09] : [NSColor colorWithWhite:0 alpha:0.05];
        NSDictionary *attrs=[storage attributesAtIndex:range.location effectiveRange:NULL];
        NSDictionary *p=attrs[CRParagraph];
        if (CRKind(p,@"code") || CRKind(p,@"source") || CRKind(p,@"opaque")) return;
        NSFont *font=attrs[NSFontAttributeName] ?: [NSFont systemFontOfSize:16*e.zoom];
        CGFloat trailingKern=[[storage attribute:NSKernAttributeName atIndex:NSMaxRange(range)-1 effectiveRange:NULL] doubleValue];
        NSRange glyphs=[lm glyphRangeForCharacterRange:range actualCharacterRange:NULL];
        NSUInteger g=glyphs.location;
        while (g<NSMaxRange(glyphs)) {
            NSRange line;
            NSRect fragment=[lm lineFragmentRectForGlyphAtIndex:g effectiveRange:&line];
            NSRange part=NSIntersectionRange(glyphs,line);
            if (!part.length) break;
            NSRect bounds=[lm boundingRectForGlyphRange:part inTextContainer:self.textContainer];
            if (NSMaxRange(part)==NSMaxRange(glyphs)) bounds.size.width=MAX(0,bounds.size.width-trailingKern);
            CGFloat baseline=fragment.origin.y+[lm locationForGlyphAtIndex:part.location].y;
            CGFloat padX=mark ? 1.5*e.zoom : 3*e.zoom, padY=mark ? 1*e.zoom : 2*e.zoom;
            NSRect chip=NSMakeRect(bounds.origin.x-padX,baseline-font.ascender-padY,bounds.size.width+2*padX,font.ascender-font.descender+2*padY);
            chip=NSOffsetRect(chip,self.textContainerOrigin.x,self.textContainerOrigin.y);
            NSBezierPath *path=[NSBezierPath bezierPathWithRoundedRect:chip xRadius:(mark ? 3 : 4)*e.zoom yRadius:(mark ? 3 : 4)*e.zoom];
            if (fill) [fill setFill];
            else if (mark) [(e.dark ? [NSColor colorWithRed:0.75 green:0.55 blue:0.10 alpha:0.45] : [NSColor colorWithRed:1 green:0.89 blue:0.45 alpha:1]) setFill];
            else if (kbd) [(e.dark ? [NSColor colorWithWhite:0.26 alpha:1] : [NSColor colorWithWhite:0.97 alpha:1]) setFill];
            else [(e.dark ? [NSColor colorWithWhite:1 alpha:0.10] : [NSColor colorWithWhite:0 alpha:0.055]) setFill];
            [path fill];
            if (kbd) {
                // A key cap: hairline border with a heavier bottom edge.
                [(e.dark ? [NSColor colorWithWhite:1 alpha:0.22] : [NSColor colorWithWhite:0 alpha:0.18]) setStroke];
                NSBezierPath *edge=[NSBezierPath bezierPathWithRoundedRect:NSInsetRect(chip,0.5,0.5) xRadius:4*e.zoom yRadius:4*e.zoom];
                edge.lineWidth=1; [edge stroke];
                [(e.dark ? [NSColor colorWithWhite:1 alpha:0.12] : [NSColor colorWithWhite:0 alpha:0.12]) setFill];
                NSRectFillUsingOperation(NSMakeRect(NSMinX(chip)+3*e.zoom,NSMaxY(chip)-1,NSWidth(chip)-6*e.zoom,1),NSCompositingOperationSourceOver);
            }
            g=NSMaxRange(part);
        }
    }];
    }
}
- (void)drawRect:(NSRect)dirty {
    CREditor *e=self.owner;
    NSRange visible=[self drawRange:dirty];
    // drawsBackground is off: paint the canvas, then one rounded slab per
    // protected block, so the chips sit under the text and stay separated by
    // the real paragraph gap.
    [self.backgroundColor setFill];
    NSRectFill(dirty);
    __block NSRect group=NSZeroRect;
    __block BOOL groupCode=NO;
    __block CGFloat groupIndent=0; // code inside a list item sits at the item's text column
    NSRect (^chipBounds)(void)=^NSRect {
        // Code gets even padding all round; its language tag sits in the
        // top-right corner instead of an empty header row.
        CGFloat pad=groupCode ? 12*e.zoom : 7*e.zoom;
        // Cards line up with the text column (and tables), inside the
        // line-fragment padding rather than the container's raw edges.
        CGFloat inset=self.textContainer.lineFragmentPadding;
        return NSMakeRect(self.textContainerOrigin.x+inset+groupIndent,group.origin.y-pad,self.textContainer.size.width-2*inset-groupIndent,group.size.height+2*pad);
    };
    void (^flush)(void)=^{
        if (NSIsEmptyRect(group)) return;
        // Set the fill here: drawing a language tag leaves its gray behind.
        [(e.dark ? [NSColor colorWithWhite:0.23 alpha:1] : [NSColor colorWithWhite:0.94 alpha:1]) setFill];
        NSRect chip=chipBounds();
        if (NSIntersectsRect(chip,dirty)) [[NSBezierPath bezierPathWithRoundedRect:chip xRadius:6*e.zoom yRadius:6*e.zoom] fill];
        group=NSZeroRect;
        groupCode=NO;
    };
    __block NSString *groupLang=nil;
    void (^flushWithLabel)(void)=^{
        // Chip first, then the language tag on top of it.
        NSRect chip=NSIsEmptyRect(group) ? NSZeroRect : chipBounds();
        flush();
        if (groupLang.length) {
            NSDictionary *attrs=@{NSFontAttributeName:[NSFont systemFontOfSize:10.5*e.zoom],NSForegroundColorAttributeName:CRMutedColor(e.dark)};
            NSSize size=[groupLang sizeWithAttributes:attrs];
            CGFloat hoverShift=_hoverCodeChip.size.height>0 && NSIntersectsRect(chip,_hoverCodeChip) ? 34*e.zoom : 0;
            [groupLang drawAtPoint:NSMakePoint(NSMaxX(chip)-size.width-10*e.zoom-hoverShift,chip.origin.y+6*e.zoom) withAttributes:attrs];
        }
        groupLang=nil;
    };
    [e drawAlertPanels:dirty range:visible];
    NSArray *rules=[e ruleRangesIn:visible];
    for (NSValue *v in CRRangesIn(self.string,visible)) {
        NSDictionary *p=CRParagraphAt(e,v.rangeValue.location);
        if ([rules containsObject:v]) { flushWithLabel(); continue; } // a rule draws as a line, not a chip
        // A blank or rich paragraph closes the block's chip.
        if (!CRKind(p,@"source") && !CRKind(p,@"code") && !CRKind(p,@"opaque")) { flushWithLabel(); continue; }
        if (CRKind(p,@"code") && [p[@"open"] length]) flushWithLabel();
        NSRange content=CRContent(self.string,v.rangeValue);
        if (!content.length && NSMaxRange(v.rangeValue)>content.location) content=NSMakeRange(content.location,1); // empty source line: the newline's own line
        NSRange glyph=[self.layoutManager glyphRangeForCharacterRange:content actualCharacterRange:NULL];
        NSRect r=glyph.length ? [self.layoutManager boundingRectForGlyphRange:glyph inTextContainer:self.textContainer] : self.layoutManager.extraLineFragmentRect;
        if (!glyph.length && !CRKind(p,@"code")) continue;
        if (r.size.height<=0) r.size.height=[self.layoutManager defaultLineHeightForFont:[NSFont monospacedSystemFontOfSize:13.5*e.zoom weight:NSFontWeightRegular]];
        r=NSOffsetRect(r,self.textContainerOrigin.x,self.textContainerOrigin.y);
        // A small vertical gap means the same block; more starts a new chip.
        if (!NSIsEmptyRect(group) && r.origin.y-NSMaxY(group) < 10*e.zoom) group=NSUnionRect(group,r);
        else {
            flushWithLabel();
            group=r;
            groupCode=CRKind(p,@"code");
            groupIndent=CRCodeListIndent(p)*e.zoom;
            if (CRKind(p,@"code")) {
                // The language tag comes from the hidden opening fence.
                NSString *open=[p[@"open"] isKindOfClass:NSString.class] ? p[@"open"] : @"";
                groupLang=[open stringByTrimmingCharactersInSet:[NSCharacterSet characterSetWithCharactersInString:@"`~ 	"]];
                groupLang=[[groupLang componentsSeparatedByCharactersInSet:NSCharacterSet.whitespaceCharacterSet] firstObject];
                if (!groupLang.length) groupLang=@"text";
            }
        }
        // A paragraph ending in a blank line closes its block.
        if (CRBlockEnding(p)) flushWithLabel();
    }
    flushWithLabel();
    for (NSValue *v in rules) {
        NSRange glyph=[self.layoutManager glyphRangeForCharacterRange:CRContent(self.string,v.rangeValue) actualCharacterRange:NULL];
        NSRect line=NSOffsetRect([self.layoutManager lineFragmentUsedRectForGlyphAtIndex:glyph.location effectiveRange:NULL],self.textContainerOrigin.x,self.textContainerOrigin.y);
        CGFloat y=round(NSMidY(line))+0.5;
        [(e.dark ? [NSColor colorWithWhite:1 alpha:0.18] : [NSColor colorWithWhite:0 alpha:0.14]) setFill];
        CGFloat inset=self.textContainer.lineFragmentPadding;
        NSRectFillUsingOperation(NSMakeRect(self.textContainerOrigin.x+inset,y-0.5,self.textContainer.size.width-2*inset,1),NSCompositingOperationSourceOver);
    }
    [self drawInlineChips:visible];
    for (NSValue *v in CRRangesIn(self.string,visible)) {
        NSRange r=v.rangeValue;
        if (!r.length || !r.location || !CRFootnoteAreaStart(e,r.location)) continue;
        NSRange glyph=[self.layoutManager glyphRangeForCharacterRange:NSMakeRange(r.location,1) actualCharacterRange:NULL];
        NSRect line=[self.layoutManager lineFragmentRectForGlyphAtIndex:glyph.location effectiveRange:NULL];
        NSParagraphStyle *style=[self.textStorage attribute:NSParagraphStyleAttributeName atIndex:r.location effectiveRange:NULL];
        CGFloat y=round(self.textContainerOrigin.y+NSMinY(line)+style.paragraphSpacingBefore/2)-0.5;
        CGFloat inset=self.textContainer.lineFragmentPadding;
        [(e.dark ? [NSColor colorWithWhite:1 alpha:0.18] : [NSColor colorWithWhite:0 alpha:0.14]) setFill];
        NSRectFillUsingOperation(NSMakeRect(self.textContainerOrigin.x+inset,y,MIN(180*e.zoom,self.textContainer.size.width/3),1),NSCompositingOperationSourceOver);
    }
    // Hover affordances paint before the glyphs: NSTextView clips its own
    // drawing to the text container, which would discard the margin handle and
    // the table buttons that sit just outside the table.
    [self drawHoverControls];
    [self drawTableControls];
    [super drawRect:dirty];
    [e drawAlertTitlesIn:visible];
    [e drawCodeLineNumbers:dirty range:visible];
    NSArray *markerRanges=CRRangesIn(self.string,visible);
    for (NSUInteger markerI=0; markerI<markerRanges.count; markerI++) {
        NSValue *v=markerRanges[markerI];
        NSDictionary *p=CRParagraphAt(e,v.rangeValue.location);
        // The first paragraph of a definition carries its number.
        if ([p[@"footnoteId"] length] && ![p[@"footnoteContinuation"] boolValue]) {
            NSRect line=[e markerLineRect:v.rangeValue];
            NSString *number=e.footnotes[p[@"footnoteId"]][@"number"] ?: @"·";
            NSDictionary *attrs=@{NSFontAttributeName:[NSFont systemFontOfSize:11.5*e.zoom weight:NSFontWeightMedium],NSForegroundColorAttributeName:NSColor.linkColor};
            [[number stringByAppendingString:@"."] drawAtPoint:NSMakePoint(self.textContainerOrigin.x+4*e.zoom,NSMidY(line)-8*e.zoom) withAttributes:attrs];
        }
        if (CRList(p) && CRQuoteOf(p)>0) {
            // A list inside a quote keeps the quote's bars beside it, joined
            // to the next line of the same quote.
            NSRange content=CRContent(self.string,v.rangeValue);
            NSRange glyph=[self.layoutManager glyphRangeForCharacterRange:content.length ? content : NSMakeRange(v.rangeValue.location,0) actualCharacterRange:NULL];
            NSRect first=[e markerLineRect:v.rangeValue];
            CGFloat top=NSMidY(first)-9.5*e.zoom, bottom=NSMidY(first)+9.5*e.zoom;
            if (glyph.length) {
                NSRange lastLine;
                [self.layoutManager lineFragmentRectForGlyphAtIndex:NSMaxRange(glyph)-1 effectiveRange:&lastLine];
                NSRect lastBounds=NSOffsetRect([self.layoutManager boundingRectForGlyphRange:lastLine inTextContainer:self.textContainer],self.textContainerOrigin.x,self.textContainerOrigin.y);
                bottom=MAX(bottom,NSMidY(lastBounds)+9.5*e.zoom);
            }
            NSInteger next=0; CGFloat reach=bottom;
            if (markerI+1<markerRanges.count) {
                NSRange nextRange=[markerRanges[markerI+1] rangeValue];
                next=CRQuoteLevel(CRParagraphAt(e,nextRange.location));
                if (next>0) reach=MAX(bottom,NSMidY([e markerLineRect:nextRange]));
            }
            NSString *alert=[p[@"alert"] isKindOfClass:NSString.class] ? p[@"alert"] : nil;
            if (alert.length) [CRAlertAccent(alert,e.dark) setFill];
            else [(e.dark ? [NSColor colorWithWhite:0.40 alpha:1] : [NSColor colorWithWhite:0.80 alpha:1]) setFill];
            for (NSInteger level=0;level<CRQuoteOf(p);level++) {
                CGFloat end=level<next ? reach : bottom;
                [[NSBezierPath bezierPathWithRoundedRect:NSMakeRect(self.textContainerOrigin.x+4*e.zoom+7+level*26*e.zoom,top,3*e.zoom,end-top) xRadius:1.5*e.zoom yRadius:1.5*e.zoom] fill];
            }
        }
        if (CRList(p) && [p[@"listContinuation"] boolValue]) continue; // a quote inside an item still draws its bar
        NSRect rect=[e markerRect:v.rangeValue];
        if (!NSIntersectsRect(rect,dirty)) continue;
        // Markers match the body size and sit on the line's vertical center,
        // so bullets and ordinals read as part of the text, not as footnotes.
        NSRect lineRect=[e markerLineRect:v.rangeValue];
        NSColor *markerColor=e.dark ? [NSColor colorWithWhite:0.70 alpha:1] : [NSColor colorWithWhite:0.35 alpha:1];
        if (CRKind(p,@"checklist")) {
            if (CROrdered(p)) {
                NSFont *font=[NSFont systemFontOfSize:16*e.zoom];
                NSString *number=[NSString stringWithFormat:@"%@.",p[@"ordinal"]];
                [number drawAtPoint:NSMakePoint(rect.origin.x-24*e.zoom,NSMidY(lineRect)-(font.ascender-font.descender)/2) withAttributes:@{NSFontAttributeName:font,NSForegroundColorAttributeName:markerColor}];
            }
            NSRect circle=NSInsetRect(rect,1,1);
            NSBezierPath *box=[NSBezierPath bezierPathWithOvalInRect:circle];
            box.lineWidth=1.4*e.zoom;
            [NSColor.systemBlueColor setStroke]; [box stroke];
            if ([p[@"checked"] boolValue]) {
                [NSColor.systemBlueColor setFill]; [box fill];
                NSDictionary *check=@{NSFontAttributeName:[NSFont systemFontOfSize:13*e.zoom weight:NSFontWeightSemibold],NSForegroundColorAttributeName:NSColor.whiteColor};
                NSSize size=[@"\u2713" sizeWithAttributes:check];
                [@"\u2713" drawAtPoint:NSMakePoint(NSMidX(circle)-size.width/2,NSMidY(circle)-size.height/2) withAttributes:check];
            }
        } else if (CRKind(p,@"bullet")) {
            // A drawn dot, not the \u2022 glyph: font bullets render tiny next to
            // CJK glyphs no matter the point size.
            CGFloat d=6.5*e.zoom;
            NSRect dot=NSMakeRect(rect.origin.x+1.5*e.zoom,NSMidY(lineRect)-d/2,d,d);
            [markerColor setFill];
            [[NSBezierPath bezierPathWithOvalInRect:dot] fill];
        } else if (CRKind(p,@"ordered")) {
            NSFont *f=[NSFont systemFontOfSize:16*e.zoom];
            [[NSString stringWithFormat:@"%@.",p[@"ordinal"] ?: @1] drawAtPoint:NSMakePoint(rect.origin.x,NSMidY(lineRect)-(f.ascender-f.descender)/2) withAttributes:@{NSFontAttributeName:f,NSForegroundColorAttributeName:markerColor}];
        }
        else if (CRKind(p,@"quote")) {
            NSString *alert = [p[@"alert"] isKindOfClass:NSString.class] ? p[@"alert"] : nil;
            if (alert.length) [CRAlertAccent(alert,e.dark) setFill];
            else [(e.dark ? [NSColor colorWithWhite:0.40 alpha:1] : [NSColor colorWithWhite:0.80 alpha:1]) setFill];
            // Bars of one blockquote connect: each spans its own wrapped
            // lines, and a following quote line of the same quote pulls it
            // across the paragraph gap, so the run reads as one edge.
            CGFloat own=NSMaxY(rect);
            NSRange content=CRContent(self.string,v.rangeValue);
            NSRange glyph=[self.layoutManager glyphRangeForCharacterRange:content actualCharacterRange:NULL];
            if (glyph.length) {
                NSRange lastLine;
                [self.layoutManager lineFragmentRectForGlyphAtIndex:NSMaxRange(glyph)-1 effectiveRange:&lastLine];
                NSRect lastBounds=[self.layoutManager boundingRectForGlyphRange:lastLine inTextContainer:self.textContainer];
                lastBounds=NSOffsetRect(lastBounds,self.textContainerOrigin.x,self.textContainerOrigin.y);
                own=MAX(own,NSMidY(lastBounds)+9.5*e.zoom);
            }
            CGFloat reach=own;
            NSInteger nextDepth=-1;
            if (markerI+1<markerRanges.count) {
                NSRange nextRange=[markerRanges[markerI+1] rangeValue];
                NSDictionary *np=CRParagraphAt(e,nextRange.location);
                NSString *nextAlert=[np[@"alert"] isKindOfClass:NSString.class] ? np[@"alert"] : nil;
                NSString *listAlert=[np[@"alert"] isKindOfClass:NSString.class] ? np[@"alert"] : nil;
                if (CRList(np) && CRQuoteOf(np)>0 && (alert==listAlert || [alert isEqual:listAlert])) {
                    reach=MAX(reach,NSMidY([e markerLineRect:nextRange]));
                    nextDepth=CRQuoteOf(np)-1;
                } else if (CRKind(np,@"quote") && (![np[@"listContinuation"] boolValue] || [np[@"listItem"] isEqual:p[@"listItem"]]) && (alert==nextAlert || [alert isEqual:nextAlert])) {
                    // Reach the next bar's middle, not its top: bar tops
                    // ride on tight glyph bounds, which jitter a point or
                    // two between lines ("11" vs "hello") and would leave
                    // hairline gaps at the joints. The overlap paints the
                    // same colour twice, so it is invisible.
                    NSRect nextRect=[e markerRect:nextRange];
                    reach=MAX(reach,NSMidY(nextRect));
                    nextDepth=[np[@"depth"] integerValue];
                }
            }
            for (NSInteger level=0;level<=[p[@"depth"] integerValue];level++) {
                CGFloat bottom=level<=nextDepth ? reach : own;
                [[NSBezierPath bezierPathWithRoundedRect:NSMakeRect(rect.origin.x+7+level*26*e.zoom,rect.origin.y,3*e.zoom,bottom-rect.origin.y) xRadius:1.5*e.zoom yRadius:1.5*e.zoom] fill];
            }
        }
    }
    // These overlays sit above the text: the image pill covers an attachment
    // and the insertion line must stay visible over glyphs.
    [self drawImagePill];
    [self drawDragOverlay];
    [self drawTableDragOverlay];
}
- (void)mouseDown:(NSEvent *)event {
    NSPoint point=[self convertPoint:event.locationInWindow fromView:nil];
    // The hover affordances sit above the text: their click never becomes a
    // selection drag or a caret move.
    [self updateHoverAtPoint:point];
    if (NSPointInRect(point, _copyRect) && [self copyHoveredCode]) return;
    if (NSPointInRect(point, _languageRect) && _languageRange.location != NSNotFound) { [self showLanguageMenu]; return; }
    if (_hoverImage.location != NSNotFound) {
        NSInteger sizeIndex=[self imageButtonAtPoint:point];
        if (sizeIndex>=0) { [self applyImageSizeAtIndex:sizeIndex]; return; }
        if (NSPointInRect(point, _imageReplaceRect)) { [self replaceImageAtHover]; return; }
        if (NSPointInRect(point, _imageCopyRect) && [self copyImageAtHover]) return;
        if (NSPointInRect(point, _imageDeleteRect) && [self deleteImageAtHover]) return;
    }
    for (NSDictionary *button in _tableButtons) {
        if (NSPointInRect(point,[button[@"rect"] rectValue]) && [self activateTableControl:[button[@"kind"] integerValue]]) return;
    }
    // A table grip starts a row/column drag; the grips sit outside the table
    // bounds, so they win over the text click.
    if (_hoverTable && (NSPointInRect(point, _tableDragRowGrip) || NSPointInRect(point, _tableDragColumnGrip))) {
        [self beginTableDragAtPoint:point];
        return;
    }
    if (NSPointInRect(point, _handleRect) && _hoverBlock.location != NSNotFound) { [self beginBlockDragAtPoint:point]; return; }
    if (self.window.firstResponder!=self) self.owner.previousResponder=self.window.firstResponder;
    NSUInteger index=[self characterIndexForInsertionAtPoint:point];
    // Only the paragraph under the pointer can own a checkbox there.
    NSRange line=[self.string paragraphRangeForRange:NSMakeRange(MIN(index,self.string.length),0)];
    NSDictionary *clicked=CRParagraphAt(self.owner,line.location);
    if (CRKind(clicked,@"checklist") && ![clicked[@"listContinuation"] boolValue] && NSPointInRect(point,[self.owner markerRect:line])) { [self.owner toggleCheckbox:line.location]; return; }
    BOOL simpleClick=event.clickCount==1 && !(event.modifierFlags & (NSEventModifierFlagShift|NSEventModifierFlagCommand|NSEventModifierFlagOption|NSEventModifierFlagControl));
    NSPoint containerPoint=NSMakePoint(point.x-self.textContainerOrigin.x,point.y-self.textContainerOrigin.y);
    // A click in the blank page below a trailing code block or figure opens
    // a line after it, so the note can always be continued (Typora, Notes).
    if (simpleClick && self.editable && self.string.length && containerPoint.y>NSMaxY([self.layoutManager usedRectForTextContainer:self.textContainer])+4) {
        [self.window makeFirstResponder:self];
        self.selectedRange=NSMakeRange(self.string.length,0);
        if ([self.owner escapeBlock:YES]) return;
    }
    NSUInteger glyph=[self.layoutManager glyphIndexForPoint:containerPoint inTextContainer:self.textContainer];
    if (simpleClick && glyph<self.layoutManager.numberOfGlyphs && NSPointInRect(containerPoint,[self.layoutManager boundingRectForGlyphRange:NSMakeRange(glyph,1) inTextContainer:self.textContainer])) {
        NSUInteger hit=[self.layoutManager characterIndexForGlyphAtIndex:glyph];
        NSDictionary *attrs=hit<self.string.length ? [self.textStorage attributesAtIndex:hit effectiveRange:NULL] : @{};
        id document=[attrs[CRMention] boolValue] ? attrs[NSLinkAttributeName] : CRKind(attrs[CRParagraph],@"page") ? CRPageTarget(attrs[CRParagraph]) : nil;
        if (document) {
            // A mention or page card opens its note on click, as in Notion; a
            // drag still selects across it.
            NSEvent *next=[self.window nextEventMatchingMask:NSEventMaskLeftMouseUp|NSEventMaskLeftMouseDragged untilDate:NSDate.distantFuture inMode:NSEventTrackingRunLoopMode dequeue:NO];
            if (next.type==NSEventTypeLeftMouseUp && [self.owner openDocumentLink:document]) {
                [self.window nextEventMatchingMask:NSEventMaskLeftMouseUp untilDate:NSDate.distantPast inMode:NSEventTrackingRunLoopMode dequeue:YES];
                return;
            }
        }
        BOOL styled=attrs[CRMathText]!=nil || attrs[CRFootnote]!=nil;
        for (NSString *key in @[CRBold,CRItalic,CRCode,NSStrikethroughStyleAttributeName,CRUnderline,CRKbd,CRHighlight,CRSup,CRSub,NSLinkAttributeName]) if (attrs[key] && ![attrs[key] isEqual:@NO]) styled=YES;
        if (styled && !attrs[CRInlineSource]) {
            // Wait for release or the first drag, leaving the event queued for
            // NSTextView when the gesture becomes a native selection drag.
            NSEvent *next=[self.window nextEventMatchingMask:NSEventMaskLeftMouseUp|NSEventMaskLeftMouseDragged untilDate:NSDate.distantFuture inMode:NSEventTrackingRunLoopMode dequeue:NO];
            if (next.type==NSEventTypeLeftMouseUp) {
                [self.window makeFirstResponder:self];
                self.selectedRange=NSMakeRange(index,0);
                NSInteger shift=(NSInteger)self.selectedRange.location-(NSInteger)index;
                NSInteger mappedHit=(NSInteger)hit+shift;
                if (mappedHit>=0 && [self.owner revealInlineAt:(NSUInteger)mappedHit]) {
                    [self.window nextEventMatchingMask:NSEventMaskLeftMouseUp untilDate:NSDate.distantPast inMode:NSEventTrackingRunLoopMode dequeue:YES];
                    return;
                }
            }
        }
    }
    if (index < self.string.length && CRKind(CRParagraphAt(self.owner,index),@"toc")) {
        // The list is a picture of the headings, so a pick jumps to the heading it
        // names. The marker itself is opened from the context menu.
        NSRange tocRange=NSMakeRange(index,1);
        for (NSValue *v in CRRanges(self.string)) if (NSLocationInRange(index,v.rangeValue)) { tocRange=v.rangeValue; break; }
        NSInteger entry=[self.owner tocEntryAtPoint:point range:tocRange];
        if (entry>=0) { [self.window makeFirstResponder:self]; [self.owner jumpToTOCEntry:entry]; return; }
    }
    if (index < self.string.length && (CRKind(CRParagraphAt(self.owner,index),@"math") || CRKind(CRParagraphAt(self.owner,index),@"diagram") || CRKind(CRParagraphAt(self.owner,index),@"image"))) { [self.window makeFirstResponder:self]; [self.owner revealSourceAt:index]; return; }
    if (index < self.string.length && CRKind(CRParagraphAt(self.owner,index),@"opaque")) { [self.owner source]; return; }
    [super mouseDown:event];
    if (simpleClick && !self.selectedRange.length) {
        if (![self.owner revealInlineAt:self.selectedRange.location]) {
            if (glyph<self.layoutManager.numberOfGlyphs && NSPointInRect(containerPoint,[self.layoutManager boundingRectForGlyphRange:NSMakeRange(glyph,1) inTextContainer:self.textContainer])) [self.owner revealInlineAt:[self.layoutManager characterIndexForGlyphAtIndex:glyph]];
        }
    }
}
- (void)copy:(id)sender {
    if (!self.selectedRange.length || ![self.owner allowRange:self.selectedRange]) return;
    [super copy:sender];
    NSMutableDictionary *doc=[[self.owner documentForRange:self.selectedRange] mutableCopy];
    NSString *selected=[self.string substringWithRange:self.selectedRange];
    // A copied LF is a paragraph boundary, including leading/trailing empty
    // paragraphs. CRContent deliberately excludes it from semantic runs.
    NSMutableArray *paragraphs=[doc[@"paragraphs"] mutableCopy];
    BOOL hasTable=NO; for (NSDictionary *paragraph in paragraphs) hasTable |= CRKind(paragraph,@"table");
    NSUInteger expected=[[selected componentsSeparatedByString:@"\n"] count];
    while (!hasTable && paragraphs.count<expected) {
        if ([selected hasPrefix:@"\n"] && paragraphs.count==expected-1) [paragraphs insertObject:CRBody() atIndex:0];
        else [paragraphs addObject:CRBody()];
    }
    doc[@"paragraphs"]=paragraphs;
    NSData *data=[NSJSONSerialization dataWithJSONObject:doc options:0 error:NULL];
    [NSPasteboard.generalPasteboard addTypes:@[CROwnPaste] owner:nil];
    [NSPasteboard.generalPasteboard setData:data forType:CROwnPaste];
    BOOL opaque=NO; for (NSDictionary *p in doc[@"paragraphs"]) opaque |= CRKind(p,@"opaque") || CRHasFootnote(p);
    if (opaque) {
        const char *json=[[NSString alloc] initWithData:data encoding:NSUTF8StringEncoding].UTF8String;
        char *source=cr_markdown_json((char *)json);
        if (source) { [NSPasteboard.generalPasteboard setString:[NSString stringWithUTF8String:source] forType:NSPasteboardTypeString]; free(source); }
    }
}
- (void)cut:(id)sender { if (![self.owner allowRange:self.selectedRange]) return; [self copy:sender]; [self insertText:@"" replacementRange:self.selectedRange]; }
// pasteParagraphs inserts a parsed fragment at the selection. A single
// inline paragraph joins the destination paragraph; blocks keep their kind.
- (void)pasteParagraphs:(NSArray *)source {
    NSArray *paragraphs=CRImportedParagraphs(source);
    BOOL opaque=NO; for (NSDictionary *p in paragraphs) opaque |= CRKind(p,@"opaque") || CRKind(p,@"table") || CRKind(p,@"source") || [p[@"footnoteId"] length];
    if (opaque && self.string.length) {
        NSData *data=[NSJSONSerialization dataWithJSONObject:@{@"paragraphs":paragraphs} options:0 error:NULL];
        cr_insert((uintptr_t)(__bridge void *)self.owner,[[NSString alloc] initWithData:data encoding:NSUTF8StringEncoding].UTF8String);
        return;
    }
    NSRange line=[self.string paragraphRangeForRange:NSMakeRange(self.selectedRange.location,0)];
    NSDictionary *destination=CRParagraphAt(self.owner,self.selectedRange.location);
    // An empty body line adopts a pasted block ("# a"); anything else keeps
    // its own kind and absorbs the pasted text inline.
    BOOL adopt=paragraphs.count==1 && CRKind(destination,@"body") && CRContent(self.string,line).length==self.selectedRange.length;
    NSDictionary *ending=self.owner.emptyParagraph;
    NSMutableAttributedString *value=[[self.owner attributedDocument:@{@"paragraphs":paragraphs} replacingRange:self.selectedRange] mutableCopy];
    if (NSMaxRange(self.selectedRange)<self.string.length) self.owner.emptyParagraph=ending;
    // Inline rich paste belongs to the destination paragraph, not the
    // source block's round-trip metadata.
    if (!opaque && !adopt && paragraphs.count==1 && value.length) [value addAttribute:CRParagraph value:destination range:NSMakeRange(0,value.length)];
    // Text after the caret stays in its own block rather than joining a
    // pasted code block as code.
    NSUInteger end=NSMaxRange(CRContent(self.string,[self.string paragraphRangeForRange:NSMakeRange(NSMaxRange(self.selectedRange),0)]));
    if (CRKind(paragraphs.lastObject,@"code") && !CRKind(destination,@"code") && NSMaxRange(self.selectedRange)<end && value.length && ![value.string hasSuffix:@"\n"]) {
        NSRange last=NSMakeRange(value.length-1,1);
        [value appendAttributedString:[[NSAttributedString alloc] initWithString:@"\n" attributes:[value attributesAtIndex:last.location effectiveRange:NULL]]];
    }
    [self.owner replace:self.selectedRange with:value selection:NSMakeRange(self.selectedRange.location+value.length,0)];
}
// Paste as Plain Text keeps Markdown punctuation literal.
- (void)pasteAsPlainText:(id)sender {
    NSPasteboard *pb=NSPasteboard.generalPasteboard;
    NSString *text=[pb stringForType:NSPasteboardTypeString];
    if (!text || CRCellAt(self.owner,self.selectedRange.location) || [self.owner replaceCodeRange:self.selectedRange withText:text]) { [self paste:sender]; return; }
    [self.owner replace:self.selectedRange with:[[NSAttributedString alloc] initWithString:text attributes:self.typingAttributes] selection:NSMakeRange(self.selectedRange.location+text.length,0)];
}
- (void)paste:(id)sender {
    NSPasteboard *pb=NSPasteboard.generalPasteboard;
    NSRange replacement=self.selectedRange;
    [self.owner prepareInlineReplacement:&replacement];
    NSTextTableBlock *cell=CRCellAt(self.owner,self.selectedRange.location);
    id source=self.selectedRange.location<self.string.length ? [self.textStorage attribute:CRInlineSource atIndex:self.selectedRange.location effectiveRange:NULL] : self.typingAttributes[CRInlineSource];
    if (source) {
        NSString *plain=[pb stringForType:NSPasteboardTypeString];
        if (!plain) {
            NSData *rtf=[pb dataForType:NSPasteboardTypeRTF];
            if (rtf) plain=[[NSAttributedString alloc] initWithRTF:rtf documentAttributes:NULL].string;
        }
        if (!plain) { NSBeep(); return; }
        // Lines would split the revealed span across paragraphs: render it
        // and paste the blocks normally.
        static BOOL rendering;
        if (!cell && !rendering && [plain rangeOfCharacterFromSet:NSCharacterSet.newlineCharacterSet].location!=NSNotFound && [self.owner finishSourceEditing]) {
            rendering=YES; [self paste:sender]; rendering=NO; return;
        }
        if (cell) plain=[[plain componentsSeparatedByCharactersInSet:NSCharacterSet.newlineCharacterSet] componentsJoinedByString:@" "];
        NSDictionary *paragraph=CRParagraphAt(self.owner,self.selectedRange.location);
        NSMutableDictionary *attrs=[@{CRParagraph:paragraph,CRInlineSource:source} mutableCopy];
        if (cell) attrs[CRTableCell]=cell;
        self.typingAttributes=[self.owner visual:attrs paragraph:paragraph];
        [self.owner replace:self.selectedRange with:[[NSAttributedString alloc] initWithString:plain attributes:self.typingAttributes] selection:NSMakeRange(self.selectedRange.location+plain.length,0)];
        return;
    }
    if (cell) {
        NSString *plain=[pb stringForType:NSPasteboardTypeString];
        if (plain) {
            if ([self.owner pasteTableText:plain]) return;
            plain=[[plain componentsSeparatedByCharactersInSet:NSCharacterSet.newlineCharacterSet] componentsJoinedByString:@" "];
            NSDictionary *paragraph=CRParagraphAt(self.owner,self.selectedRange.location);
            NSDictionary *attrs=[self.owner visual:@{CRParagraph:paragraph,CRTableCell:cell} paragraph:paragraph];
            [self.owner replace:self.selectedRange with:[[NSAttributedString alloc] initWithString:plain attributes:attrs] selection:NSMakeRange(self.selectedRange.location+plain.length,0)];
        }
        return;
    }
    NSString *codePlain=[pb stringForType:NSPasteboardTypeString];
    // A URL pasted over selected prose links the selection (Typora).
    NSString *url=[codePlain stringByTrimmingCharactersInSet:NSCharacterSet.whitespaceAndNewlineCharacterSet];
    NSDictionary *target=CRParagraphAt(self.owner,self.selectedRange.location);
    if (self.selectedRange.length && url.length && [[NSRegularExpression regularExpressionWithPattern:@"^(https?|mailto):\\S+$" options:NSRegularExpressionCaseInsensitive error:NULL] firstMatchInString:url options:0 range:NSMakeRange(0,url.length)]
        && ![[self.string substringWithRange:self.selectedRange] containsString:@"\n"] && !CRKind(target,@"code") && !CRKind(target,@"source") && !CRKind(target,@"opaque")) {
        [self.owner format:[@"link:" stringByAppendingString:url]];
        return;
    }
    if (!codePlain) {
        NSData *rtf=[pb dataForType:NSPasteboardTypeRTF];
        if (rtf) codePlain=[[NSAttributedString alloc] initWithRTF:rtf documentAttributes:NULL].string;
    }
    if (codePlain && [self.owner replaceCodeRange:replacement withText:codePlain]) return;
    // Raw source (an HTML block, an unsupported construct) takes text as is.
    if (codePlain && CRKind(target,@"source") && !CRStaleMarker(target,@"")) { [self insertText:codePlain replacementRange:self.selectedRange]; return; }
    NSData *own=[pb dataForType:CROwnPaste];
    if (own) { [self pasteParagraphs:[NSJSONSerialization JSONObjectWithData:own options:0 error:NULL][@"paragraphs"]]; return; }
    // Smart paste: clipboard images, copied files, and HTML become Markdown
    // through the Go handlers; whatever they decline falls through.
    NSData *imageData=[pb dataForType:NSPasteboardTypePNG] ?: [pb dataForType:NSPasteboardTypeTIFF];
    if (imageData && self.owner.callback) {
        NSBitmapImageRep *rep=[NSBitmapImageRep imageRepWithData:imageData];
        NSData *png=[rep representationUsingType:NSBitmapImageFileTypePNG properties:@{}];
        char *markdown=png ? cr_paste_image(self.owner.callback,[png base64EncodedStringWithOptions:0].UTF8String,"pasted-image.png") : NULL;
        if (markdown) { NSString *value=[NSString stringWithUTF8String:markdown]; free(markdown); [self.owner insertMarkdownText:value]; return; }
    }
    NSArray *paths=[pb propertyListForType:NSFilenamesPboardType];
    if (![paths isKindOfClass:NSArray.class]) paths=nil;
    if (!paths.count) { NSURL *url=[NSURL URLFromPasteboard:pb]; if (url.isFileURL) paths=@[url.path]; }
    if (paths.count && self.owner.callback) {
        NSData *data=[NSJSONSerialization dataWithJSONObject:paths options:0 error:NULL];
        char *markdown=data ? cr_drop_files(self.owner.callback,[[NSString alloc] initWithData:data encoding:NSUTF8StringEncoding].UTF8String) : NULL;
        if (markdown) { NSString *value=[NSString stringWithUTF8String:markdown]; free(markdown); [self.owner insertMarkdownText:value]; return; }
    }
    NSString *html=[pb stringForType:NSPasteboardTypeHTML];
    if (html.length) {
        char *markdown=cr_html_to_markdown(html.UTF8String);
        if (markdown) { NSString *value=[NSString stringWithUTF8String:markdown]; free(markdown); if (value.length) { [self.owner insertMarkdownText:value]; return; } }
    }
    NSData *rtfd=[pb dataForType:NSPasteboardTypeRTFD];
    if (rtfd) { [self.owner source]; NSBeep(); return; }
    NSData *rtf=[pb dataForType:NSPasteboardTypeRTF];
    if (rtf) {
        NSAttributedString *rich=[[NSAttributedString alloc] initWithRTF:rtf documentAttributes:NULL];
        __block BOOL unsupported=NO;
        [rich enumerateAttributesInRange:NSMakeRange(0,rich.length) options:0 usingBlock:^(NSDictionary *attrs, NSRange range, BOOL *stop) {
            NSParagraphStyle *style=attrs[NSParagraphStyleAttributeName];
            if (attrs[NSAttachmentAttributeName] || style.textBlocks.count || style.textLists.count || (attrs[NSSuperscriptAttributeName] && [attrs[NSSuperscriptAttributeName] integerValue]) || [attrs[NSUnderlineStyleAttributeName] integerValue] || [attrs[NSBaselineOffsetAttributeName] doubleValue]) unsupported=YES;
        }];
        if (unsupported || !rich) { [self.owner source]; NSBeep(); return; }
        NSMutableAttributedString *clean=[[NSMutableAttributedString alloc] initWithString:rich.string];
        NSDictionary *p=CRFresh(CRParagraphAt(self.owner,self.selectedRange.location));
        [rich enumerateAttributesInRange:NSMakeRange(0,rich.length) options:0 usingBlock:^(NSDictionary *attrs, NSRange range, BOOL *stop) {
            NSMutableDictionary *a=[@{CRParagraph:p} mutableCopy];
            NSFontTraitMask traits=[[NSFontManager sharedFontManager] traitsOfFont:attrs[NSFontAttributeName] ?: [NSFont systemFontOfSize:17]];
            if (traits & NSBoldFontMask) a[CRBold]=@YES;
            if (traits & NSItalicFontMask) a[CRItalic]=@YES;
            if (attrs[NSStrikethroughStyleAttributeName]) a[NSStrikethroughStyleAttributeName]=attrs[NSStrikethroughStyleAttributeName];
            if (attrs[NSLinkAttributeName]) a[NSLinkAttributeName]=attrs[NSLinkAttributeName];
            [clean setAttributes:[self.owner visual:a paragraph:p] range:range];
        }];
        [self.owner replace:self.selectedRange with:clean selection:NSMakeRange(self.selectedRange.location+clean.length,0)];
        return;
    }
    NSString *text=[pb stringForType:NSPasteboardTypeString];
    if (text.length && !CRCellAt(self.owner,self.selectedRange.location)) {
        // Plain text is Markdown, as in Typora: "# a" pastes a heading.
        char *json=cr_parse_markdown((char *)text.UTF8String);
        NSDictionary *document=json ? CRJSON(json) : nil;
        if (json) free(json);
        if ([document[@"paragraphs"] count]) { [self pasteParagraphs:document[@"paragraphs"]]; return; }
    }
    if (text) {
        NSDictionary *attrs=self.typingAttributes;
        if (CRCellAt(self.owner,self.selectedRange.location)) text=[[text componentsSeparatedByCharactersInSet:NSCharacterSet.newlineCharacterSet] componentsJoinedByString:@" "];
        [self.owner replace:self.selectedRange with:[[NSAttributedString alloc] initWithString:text attributes:attrs] selection:NSMakeRange(self.selectedRange.location+text.length,0)];
    }
    else { [self.owner source]; NSBeep(); }
}
// File drops become Markdown through the Go importer, inserted at the drop
// point. Declined files return NO so the window can still open them.
- (NSDragOperation)draggingEntered:(id<NSDraggingInfo>)info {
    NSPasteboard *pb=info.draggingPasteboard;
    if ([pb.types containsObject:NSFilenamesPboardType] || [pb.types containsObject:NSPasteboardTypeFileURL]) return NSDragOperationCopy;
    return [super draggingEntered:info];
}
- (NSDragOperation)draggingUpdated:(id<NSDraggingInfo>)info { return [self draggingEntered:info]; }
- (BOOL)performDragOperation:(id<NSDraggingInfo>)info {
    NSPasteboard *pb=info.draggingPasteboard;
    NSArray *paths=[pb propertyListForType:NSFilenamesPboardType];
    if (![paths isKindOfClass:NSArray.class] || !paths.count) {
        NSURL *url=[NSURL URLFromPasteboard:pb];
        paths=url.isFileURL ? @[url.path] : nil;
    }
    if (!paths.count || !self.owner.callback) return [super performDragOperation:info];
    NSData *data=[NSJSONSerialization dataWithJSONObject:paths options:0 error:NULL];
    char *markdown=data ? cr_drop_files(self.owner.callback,[[NSString alloc] initWithData:data encoding:NSUTF8StringEncoding].UTF8String) : NULL;
    if (!markdown) return NO;
    NSString *value=[NSString stringWithUTF8String:markdown]; free(markdown);
    NSPoint point=[self convertPoint:info.draggingLocation fromView:nil];
    self.selectedRange=NSMakeRange([self characterIndexForInsertionAtPoint:point],0);
    [self.owner insertMarkdownText:value];
    return YES;
}
#include "structure_controls_darwin.h"
#include "hover_controls_darwin.h"
@end

static NSView *CRUnderlyingResponder(NSView *view, NSView *excluded) {
    if (view==excluded) return nil;
    for (NSView *child in view.subviews) { NSView *found=CRUnderlyingResponder(child,excluded); if (found) return found; }
    return view.acceptsFirstResponder ? view : nil;
}
static CREditor *CRE(uintptr_t ptr) { return (__bridge CREditor *)(void *)ptr; }
static NSDictionary *CRJSON(const char *json) { return [NSJSONSerialization JSONObjectWithData:[[NSString stringWithUTF8String:json] dataUsingEncoding:NSUTF8StringEncoding] options:0 error:NULL]; }
uintptr_t cr_new(uintptr_t window, uintptr_t callback) {
    if (!window || !NSThread.isMainThread) return 0;
    NSWindow *w=(__bridge NSWindow *)(void *)window;
    if (!w.contentView) return 0;
    CREditor *e=[CREditor new]; e.window=w; e.callback=callback; e.zoom=1; e.emptyParagraph=CRBody(); e.undo=[NSUndoManager new];
    e.scroll=[[NSScrollView alloc] initWithFrame:NSMakeRect(0,0,600,400)]; e.scroll.hidden=YES; e.scroll.hasVerticalScroller=YES; e.scroll.autohidesScrollers=YES; e.scroll.borderType=NSNoBorder;
    e.text=[[CRTextView alloc] initWithFrame:e.scroll.contentView.bounds]; e.text.owner=e; e.text.delegate=e;
    e.text.richText=YES; e.text.importsGraphics=NO; e.text.allowsUndo=YES; e.text.verticallyResizable=YES; e.text.horizontallyResizable=NO;
    e.text.minSize=NSMakeSize(0,0); e.text.maxSize=NSMakeSize(CGFLOAT_MAX,CGFLOAT_MAX); e.text.autoresizingMask=NSViewWidthSizable;
    e.text.textContainer.containerSize=NSMakeSize(e.scroll.contentSize.width,CGFLOAT_MAX); e.text.textContainer.widthTracksTextView=YES;
    e.text.textContainerInset=NSMakeSize(36,22); e.text.automaticQuoteSubstitutionEnabled=NO; e.text.automaticDashSubstitutionEnabled=NO;
    e.text.automaticLinkDetectionEnabled=NO; e.text.automaticDataDetectionEnabled=NO; e.text.automaticTextReplacementEnabled=NO;
    // Link ink and underline come from visual:, so a mention chip or a
    // colored link can look like itself; AppKit only adds the hand cursor.
    e.text.linkTextAttributes=@{NSCursorAttributeName:NSCursor.pointingHandCursor};
    // Markdown is literal: no silent autocorrect or spaces around pasted words.
    e.text.automaticSpellingCorrectionEnabled=NO; e.text.smartInsertDeleteEnabled=NO;
    e.text.drawsBackground=NO; // CRTextView.drawRect paints canvas and chips itself
    e.draftGroup=NSMakeRange(NSNotFound,0); e.dirty=NSMakeRange(NSNotFound,0); e.ruleCaret=NSMakeRange(NSNotFound,0);
    e.lastCaret=NSNotFound;
    [e.text registerForDraggedTypes:@[NSFilenamesPboardType,NSPasteboardTypeFileURL]];
    e.mathCache=[NSMutableDictionary new]; e.imageCache=[NSMutableDictionary new]; e.imagePending=[NSMutableDictionary new]; e.imageAttachments=[NSMutableDictionary new]; e.imageSpecForURL=[NSMutableDictionary new]; e.tocSignature=[NSString string];
    e.text.typingAttributes=@{CRParagraph:CRBody()}; e.scroll.documentView=e.text; [w.contentView addSubview:e.scroll positioned:NSWindowAbove relativeTo:nil];
    [e restyle];
    return (uintptr_t)(__bridge_retained void *)e;
}
void cr_close(uintptr_t ptr) { CREditor *e=(__bridge_transfer CREditor *)(void *)ptr; [e slashClose]; [e emojiMenuClose]; [e linkMenuClose]; [e formatBarHide]; [e.text clearHover]; [e.text clearDrag]; [e.text closeFootnoteCard]; [e clearFootnoteBack]; e.callback=0; e.text.delegate=nil; [e.scroll removeFromSuperview]; [e.undo removeAllActions]; }
void cr_load(uintptr_t ptr,const char *json) { CREditor *e=CRE(ptr); [e slashClose]; [e emojiMenuClose]; [e linkMenuClose]; [e formatBarHide]; [e.text clearHover]; [e.text clearDrag]; [e clearFootnoteBack]; e.documentRevision++; e.suppress=YES; e.tocSignature=nil; e.mathSignature=nil; NSAttributedString *value=[e attributedDocument:CRJSON(json)]; [e.text.textStorage setAttributedString:value]; e.text.selectedRange=NSMakeRange(0,0); e.text.typingAttributes=@{CRParagraph: value.length ? [value attribute:CRParagraph atIndex:0 effectiveRange:NULL] : e.emptyParagraph ?: CRBody()}; [e.undo removeAllActions]; e.suppress=NO; [e restyle]; [e refreshMath]; [e refreshTOC]; }
char *cr_snapshot(uintptr_t ptr,int *marked) { CREditor *e=CRE(ptr); *marked=e.text.hasMarkedText; if (*marked) return NULL; NSData *data=[NSJSONSerialization dataWithJSONObject:[e document] options:0 error:NULL]; if (!data) return NULL; return strdup([[NSString alloc] initWithData:data encoding:NSUTF8StringEncoding].UTF8String); }
void cr_format(uintptr_t ptr,const char *command) { [CRE(ptr) format:[NSString stringWithUTF8String:command]]; }
void cr_insert(uintptr_t ptr,const char *json) {
    CREditor *e=CRE(ptr); NSRange selection=e.text.selectedRange;
    if (!e.text.editable || e.text.hasMarkedText || ![e allowRange:selection]) return;
    NSTextTableBlock *destination=CRCellAt(e,selection.location);
    if (destination) selection=NSMakeRange(NSMaxRange(CRTableRangeNear(e,destination.table,selection.location)),0);
    BOOL endsAtEOF=NSMaxRange(selection)==e.text.string.length;
    NSMutableDictionary *doc=[CRJSON(json) mutableCopy];
    doc[@"paragraphs"]=CRImportedParagraphs(doc[@"paragraphs"]);
    NSDictionary *container=CRParagraphAt(e,e.text.selectedRange.location);
    doc[@"paragraphs"]=[e footnoteFragment:doc[@"paragraphs"] container:container];
    NSDictionary *ending=e.emptyParagraph;
    NSMutableAttributedString *insert=[[e attributedDocument:doc replacingRange:selection] mutableCopy];
    e.emptyParagraph=ending;
    NSDictionary *boundary=CRInFootnote(CRBody(),container);
    NSDictionary *attrs=[e visual:@{CRParagraph:boundary} paragraph:boundary];
    if (selection.location && [e.text.string characterAtIndex:selection.location-1]!='\n') [insert insertAttributedString:[[NSAttributedString alloc] initWithString:@"\n" attributes:attrs] atIndex:0];
    // A table already ends in its last cell's line break.
    BOOL endsInTable=insert.length && [insert attribute:CRTableCell atIndex:insert.length-1 effectiveRange:NULL]!=nil;
    if (!endsInTable) [insert appendAttributedString:[[NSAttributedString alloc] initWithString:@"\n" attributes:attrs]];
    __block NSUInteger tableStart=NSNotFound;
    [insert enumerateAttribute:CRTableCell inRange:NSMakeRange(0,insert.length) options:0 usingBlock:^(id cell, NSRange range, BOOL *stop) { if (cell) { tableStart=range.location; *stop=YES; } }];
    // An inserted table selects its first header so typing replaces it.
    NSRange caret=NSMakeRange(selection.location+(tableStart==NSNotFound ? insert.length : tableStart),0);
    if (tableStart!=NSNotFound) caret.length=CRContent(insert.string,[insert.string paragraphRangeForRange:NSMakeRange(tableStart,0)]).length;
    e.suppress=YES;
    [e replace:selection with:insert selection:caret];
    e.text.typingAttributes=attrs; e.emptyParagraph=endsAtEOF ? boundary : ending; e.suppress=NO;
    [e updateTypingParagraph]; [e notify];
}
// CRUpdateInset caps the reading measure: past ~47 CJK chars per line the eye
// loses the return sweep, so on wide panes the text column centers itself
// (Typora-style) instead of stretching edge to edge.
static void CRUpdateInset(CREditor *e) {
    CGFloat maxW=760*e.zoom;
    CGFloat width=e.scroll.contentSize.width;
    CGFloat x=MAX(36*e.zoom,(width-maxW)/2);
    NSSize inset=NSMakeSize(x,CRTopInset(e));
    if (!NSEqualSizes(e.text.textContainerInset,inset)) e.text.textContainerInset=inset;
}
void cr_frame(uintptr_t ptr,double x,double y,double w,double h) { CREditor *e=CRE(ptr); NSView *parent=e.scroll.superview; CGFloat top=parent.isFlipped ? y : parent.bounds.size.height-y-h; NSRect frame=NSMakeRect(x,top,MAX(0,w),MAX(0,h)); if (!NSEqualRects(frame,e.scroll.frame)) { e.scroll.frame=frame; CRUpdateInset(e); } }
void cr_visible(uintptr_t ptr,int visible) { CREditor *e=CRE(ptr); if (!visible) { [e slashClose]; [e formatBarHide]; } if (!visible && e.window.firstResponder==e.text) cr_focus(ptr,0); e.scroll.hidden=!visible; }
void cr_base_dir(uintptr_t ptr,const char *path) { CRE(ptr).baseDir=[NSString stringWithUTF8String:path]; }
void cr_font(uintptr_t ptr,const char *body,const char *mono) {
    CREditor *e=CRE(ptr);
    NSString *bodyName=body&&strlen(body)?[NSString stringWithUTF8String:body]:nil;
    NSString *monoName=mono&&strlen(mono)?[NSString stringWithUTF8String:mono]:nil;
    if ((e.bodyFont==bodyName || [e.bodyFont isEqualToString:bodyName]) && (e.monoFont==monoName || [e.monoFont isEqualToString:monoName])) return;
    e.bodyFont=bodyName; e.monoFont=monoName;
    [e restyle];
    e.tocSignature=nil; [e refreshTOC]; // the TOC figure's labels use the body font
}
void cr_style(uintptr_t ptr,double zoom,int dark,int editable) {
    CREditor *e=CRE(ptr); zoom=MAX(0.5,MIN(3,zoom));
    if (e.zoom==zoom && e.dark==dark && e.text.editable==editable) return;
    [e slashClose];
    [e emojiMenuClose];
    [e linkMenuClose];
    [e formatBarHide];
    BOOL figures=e.zoom!=zoom || e.dark!=dark;
    e.zoom=zoom; e.dark=dark; e.text.editable=editable;
    CRUpdateInset(e);
    if (figures && !e.text.hasMarkedText) {
        BOOL wasSuppress=e.suppress; e.suppress=YES;
        NSAttributedString *copy=[e.text.textStorage copy];
        [copy enumerateAttribute:NSAttachmentAttributeName inRange:NSMakeRange(0,copy.length) options:0 usingBlock:^(id attachment, NSRange range, BOOL *stop) {
            if (!attachment) return;
            NSDictionary *paragraph=CRParagraphAt(e,range.location);
            if (!CRKind(paragraph,@"image") && !CRKind(paragraph,@"diagram") && !CRKind(paragraph,@"toc") && !CRKind(paragraph,@"page")) return;
            NSAttributedString *figure=CRKind(paragraph,@"toc") ? [e tocFigure:paragraph] : CRKind(paragraph,@"page") ? [e pageFigure:paragraph] : [e figure:paragraph];
            NSTextAttachment *updated=figure.length ? [figure attribute:NSAttachmentAttributeName atIndex:0 effectiveRange:NULL] : nil;
            if (updated) [e.text.textStorage addAttribute:NSAttachmentAttributeName value:updated range:range];
        }];
        e.suppress=wasSuppress;
    }
    [e restyle];
    if (figures) { e.tocSignature=nil; [e refreshMath]; [e refreshTOC]; }
}
void cr_focus(uintptr_t ptr,int focus) { CREditor *e=CRE(ptr); if (focus) { if (e.window.firstResponder!=e.text) e.previousResponder=e.window.firstResponder; [e.window makeFirstResponder:e.text]; } else if (e.window.firstResponder==e.text) { NSResponder *previous=e.previousResponder; if (!previous || previous==e.text) previous=CRUnderlyingResponder(e.window.contentView,e.scroll) ?: e.window.contentView; [e.window makeFirstResponder:previous]; } }

static int CRExitCode;
int cr_test_loop(void) { @autoreleasepool {
    [NSApplication sharedApplication];
    NSMutableArray *savedClipboard=[NSMutableArray array];
    for (NSPasteboardItem *item in NSPasteboard.generalPasteboard.pasteboardItems) {
        NSPasteboardItem *copy=[NSPasteboardItem new];
        for (NSPasteboardType type in item.types) { NSData *data=[item dataForType:type]; if (data) [copy setData:data forType:type]; }
        [savedClipboard addObject:copy];
    }
    NSApp.activationPolicy=NSApplicationActivationPolicyAccessory; [NSApp run];
    [NSPasteboard.generalPasteboard clearContents];
    if (savedClipboard.count) [NSPasteboard.generalPasteboard writeObjects:savedClipboard];
    return CRExitCode;
} }
void cr_test_exit(int code) { dispatch_async(dispatch_get_main_queue(),^{ CRExitCode=code; [NSApp stop:nil]; NSEvent *event=[NSEvent otherEventWithType:NSEventTypeApplicationDefined location:NSZeroPoint modifierFlags:0 timestamp:0 windowNumber:0 context:nil subtype:0 data1:0 data2:0]; [NSApp postEvent:event atStart:NO]; }); }
void cr_test_main(uintptr_t handle) { dispatch_sync(dispatch_get_main_queue(),^{ @autoreleasepool { cr_test_callback(handle); } }); }
int cr_test_finish_event(uintptr_t ptr) {
    if (NSThread.isMainThread) return 0;
    CREditor *e=CRE(ptr);
    dispatch_semaphore_t completed=dispatch_semaphore_create(0);
    __block id observer=nil;
    dispatch_sync(dispatch_get_main_queue(),^{
        observer=[NSNotificationCenter.defaultCenter addObserverForName:NSUndoManagerDidCloseUndoGroupNotification object:e.undo queue:nil usingBlock:^(NSNotification *note) {
            dispatch_async(dispatch_get_main_queue(),^{ if (e.undo.groupingLevel==0) dispatch_semaphore_signal(completed); });
        }];
        [e.text breakUndoCoalescing];
        if (e.undo.groupingLevel==0) dispatch_semaphore_signal(completed);
        // Dispatch callbacks alone do not advance NSApplication's input cycle.
        // Wake nextEvent so AppKit can close its automatic undo event group.
        NSEvent *event=[NSEvent otherEventWithType:NSEventTypeApplicationDefined location:NSZeroPoint modifierFlags:0 timestamp:0 windowNumber:0 context:nil subtype:0 data1:0 data2:0];
        [NSApp postEvent:event atStart:NO];
    });
    long result=dispatch_semaphore_wait(completed,dispatch_time(DISPATCH_TIME_NOW,2*NSEC_PER_SEC));
    dispatch_sync(dispatch_get_main_queue(),^{ [NSNotificationCenter.defaultCenter removeObserver:observer]; });
    return result==0;
}
uintptr_t cr_test_window(void) { NSWindow *window=[[NSWindow alloc] initWithContentRect:NSMakeRect(0,0,720,500) styleMask:NSWindowStyleMaskTitled backing:NSBackingStoreBuffered defer:NO]; window.releasedWhenClosed=NO; return (uintptr_t)(__bridge_retained void *)window; }
void cr_test_close_window(uintptr_t ptr) { NSWindow *window=(__bridge_transfer NSWindow *)(void *)ptr; [window close]; }
void cr_test_select(uintptr_t ptr,int start,int length) { CRE(ptr).text.selectedRange=NSMakeRange(start,length); }
char *cr_test_text(uintptr_t ptr) { return strdup(CRE(ptr).text.string.UTF8String); }
char *cr_test_clipboard_text(void) { return strdup(([NSPasteboard.generalPasteboard stringForType:NSPasteboardTypeString] ?: @"").UTF8String); }
char *cr_test_code_controls(uintptr_t ptr) {
    CREditor *editor=CRE(ptr);
    NSDictionary *state=[editor respondsToSelector:@selector(codeControlState)] ? [editor codeControlState] : @{};
    NSData *json=[NSJSONSerialization dataWithJSONObject:state options:0 error:NULL];
    return strdup([[NSString alloc] initWithData:json encoding:NSUTF8StringEncoding].UTF8String);
}
char *cr_test_footnotes(uintptr_t ptr) {
    CREditor *e=CRE(ptr);
    NSMutableArray *references=[NSMutableArray array], *definitions=[NSMutableArray array];
    [e.text.textStorage enumerateAttributesInRange:NSMakeRange(0,e.text.string.length) options:0 usingBlock:^(NSDictionary *attrs, NSRange range, BOOL *stop) {
        id run=attrs[CRFootnote];
        if (![run isKindOfClass:NSDictionary.class]) return;
        [references addObject:@{@"label":run[@"footnote"] ?: @"",@"location":@(range.location),@"length":@(range.length),@"number":attrs[@"CorraFootnoteNumber"] ?: @"",@"tooltip":attrs[NSToolTipAttributeName] ?: @"",@"attachment":attrs[NSAttachmentAttributeName] ? @YES : @NO}];
    }];
    for (NSValue *value in CRRanges(e.text.string)) {
        NSDictionary *p=CRParagraphAt(e,value.rangeValue.location);
        if ([p[@"footnoteId"] length] && ![p[@"footnoteContinuation"] boolValue]) [definitions addObject:@{@"label":p[@"footnoteId"],@"location":@(value.rangeValue.location),@"length":@(CRContent(e.text.string,value.rangeValue).length)}];
    }
    NSData *json=[NSJSONSerialization dataWithJSONObject:@{@"references":references,@"definitions":definitions,@"document":[e document]} options:0 error:NULL];
    return strdup([[NSString alloc] initWithData:json encoding:NSUTF8StringEncoding].UTF8String);
}
void cr_test_selection(uintptr_t ptr,int *start,int *length) { NSRange r=CRE(ptr).text.selectedRange; *start=(int)r.location; *length=(int)r.length; }
char *cr_test_images(uintptr_t ptr) {
    CREditor *e=CRE(ptr);
    NSMutableArray *figures=[NSMutableArray array];
    for (NSValue *value in [e imageFigureRanges]) {
        NSRange range=CRContent(e.text.string,value.rangeValue);
        NSTextAttachment *att=[e.text.textStorage attribute:NSAttachmentAttributeName atIndex:range.location effectiveRange:NULL];
        if (!att) continue;
        NSDictionary *p=CRParagraphAt(e,value.rangeValue.location);
        NSString *raw=[p[@"raw"] isKindOfClass:NSString.class] ? p[@"raw"] : @"";
        [figures addObject:@{@"url":CRImageRawURL(raw) ?: @"",@"spec":CRImageRawSpec(raw) ?: @"",@"width":@(att.bounds.size.width),@"height":@(att.bounds.size.height)}];
    }
    NSData *json=[NSJSONSerialization dataWithJSONObject:figures options:0 error:NULL];
    return strdup([[NSString alloc] initWithData:json encoding:NSUTF8StringEncoding].UTF8String);
}

// cr_test_toc reports the [TOC] figures of the document: the entries they list
// and the size they are drawn at.
char *cr_test_toc(uintptr_t ptr) {
    CREditor *e=CRE(ptr);
    NSMutableArray *tocs=[NSMutableArray array];
    for (NSValue *value in [e tocFigureRanges]) {
        NSRange range=CRContent(e.text.string,value.rangeValue);
        NSTextAttachment *att=[e.text.textStorage attribute:NSAttachmentAttributeName atIndex:range.location effectiveRange:NULL];
        if (!att) continue;
        [tocs addObject:@{@"entries":[e tocEntriesFromText],@"width":@(att.bounds.size.width),@"height":@(att.bounds.size.height)}];
    }
    NSData *json=[NSJSONSerialization dataWithJSONObject:tocs options:0 error:NULL];
    return strdup([[NSString alloc] initWithData:json encoding:NSUTF8StringEncoding].UTF8String);
}
// cr_test_slash reports the open state, filter, highlighted row and visible
// palette entries of the slash menu.
char *cr_test_slash(uintptr_t ptr) {
    NSData *json=[NSJSONSerialization dataWithJSONObject:[CRE(ptr) slashState] options:0 error:NULL];
    NSString *value=json ? [[NSString alloc] initWithData:json encoding:NSUTF8StringEncoding] : nil;
    return strdup(value.UTF8String ?: "{}");
}
int cr_test_slash_shot(uintptr_t ptr,const char *path) {
    return [CRE(ptr) slashSnapshotTo:[NSString stringWithUTF8String:path]] ? 1 : 0;
}
char *cr_test_format_bar(uintptr_t ptr) {
    NSData *json=[NSJSONSerialization dataWithJSONObject:[CRE(ptr) formatBarState] options:0 error:NULL];
    NSString *value=json ? [[NSString alloc] initWithData:json encoding:NSUTF8StringEncoding] : nil;
    return strdup(value.UTF8String ?: "{}");
}
int cr_test_image_copy(uintptr_t ptr) {
    return [CRE(ptr).text copyImageAtHover] ? 1 : 0;
}
int cr_test_image_delete(uintptr_t ptr) {
    return [CRE(ptr).text deleteImageAtHover] ? 1 : 0;
}
// cr_test_image_replace swaps the first figure's file for a given path, the
// way the replace button does once NSOpenPanel returns.
int cr_test_image_replace(uintptr_t ptr,const char *path) {
    CREditor *e=CRE(ptr);
    NSArray *figures=[e imageFigureRanges];
    if (!figures.count) return 0;
    return [e replaceImageAtLocation:[[figures firstObject] rangeValue].location withURL:[NSString stringWithUTF8String:path]] ? 1 : 0;
}
// cr_test_click_image_button sends a real mouse event to a pill button, so
// the dispatch a physical click takes is the one under test: 0..4 are the
// size presets, 5 copy, 6 delete.
int cr_test_click_image_button(uintptr_t ptr,int index) {
    CREditor *e=CRE(ptr);
    NSArray *figures=[e imageFigureRanges];
    if (!figures.count) return 0;
    NSRange range=[[figures firstObject] rangeValue];
    NSRange content=CRContent(e.text.string,range);
    NSRange glyph=[e.text.layoutManager glyphRangeForCharacterRange:content actualCharacterRange:NULL];
    if (!glyph.length) return 0;
    NSRect bounds=[e.text.layoutManager boundingRectForGlyphRange:glyph inTextContainer:e.text.textContainer];
    bounds=NSOffsetRect(bounds,e.text.textContainerOrigin.x,e.text.textContainerOrigin.y);
    [e.text updateHoverAtPoint:NSMakePoint(NSMidX(bounds),NSMidY(bounds))];
    NSRect button=[e.text imagePillButtonRect:index];
    if (!button.size.width) return 0;
    NSPoint point=[e.text convertPoint:NSMakePoint(NSMidX(button),NSMidY(button)) toView:nil];
    NSEvent *event=[NSEvent mouseEventWithType:NSEventTypeLeftMouseDown location:point modifierFlags:0 timestamp:0 windowNumber:e.window.windowNumber context:nil eventNumber:0 clickCount:1 pressure:1];
    [e.text mouseDown:event];
    return 1;
}
int cr_test_click_table_button(uintptr_t ptr,int kind) {
    CREditor *e=CRE(ptr);
    if (!cr_test_hover_table(ptr)) return 0;
    NSRect button=[e.text tableButtonRectForKind:kind];
    if (!button.size.width) return 0;
    NSPoint point=[e.text convertPoint:NSMakePoint(NSMidX(button),NSMidY(button)) toView:nil];
    NSEvent *event=[NSEvent mouseEventWithType:NSEventTypeLeftMouseDown location:point modifierFlags:0 timestamp:0 windowNumber:e.window.windowNumber context:nil eventNumber:0 clickCount:1 pressure:1];
    [e.text mouseDown:event];
    return 1;
}
int cr_test_clipboard_image(void) {
    NSPasteboard *pasteboard = [NSPasteboard generalPasteboard];
    return [pasteboard availableTypeFromArray:@[NSPasteboardTypePNG, NSPasteboardTypeTIFF]] ? 1 : 0;
}
int cr_test_hover_table(uintptr_t ptr) {
    CREditor *e=CRE(ptr);
    __block NSRange cell=NSMakeRange(NSNotFound,0);
    [e.text.textStorage enumerateAttribute:CRTableCell inRange:NSMakeRange(0,e.text.string.length) options:0 usingBlock:^(id block,NSRange range,BOOL *stop) { if (block) { cell=range; *stop=YES; } }];
    if (cell.location==NSNotFound) return 0;
    NSRange glyph=[e.text.layoutManager glyphRangeForCharacterRange:cell actualCharacterRange:NULL];
    if (!glyph.length) return 0;
    NSRect rect=[e.text.layoutManager boundingRectForGlyphRange:glyph inTextContainer:e.text.textContainer];
    rect=NSOffsetRect(rect,e.text.textContainerOrigin.x,e.text.textContainerOrigin.y);
    [e.text updateHoverAtPoint:NSMakePoint(NSMidX(rect),NSMidY(rect))];
    return [[e.text hoverState][@"table"] boolValue] ? 1 : 0;
}
int cr_test_table_button(uintptr_t ptr,int kind) {
    return [CRE(ptr).text activateTableControl:kind] ? 1 : 0;
}
int cr_test_hover_footnote(uintptr_t ptr) {
    CREditor *e=CRE(ptr);
    __block NSRange found=NSMakeRange(NSNotFound,0);
    [e.text.textStorage enumerateAttribute:CRFootnote inRange:NSMakeRange(0,e.text.string.length) options:0 usingBlock:^(id run,NSRange range,BOOL *stop) { if (run) { found=range; *stop=YES; } }];
    if (found.location==NSNotFound) return 0;
    NSRange glyph=[e.text.layoutManager glyphRangeForCharacterRange:found actualCharacterRange:NULL];
    if (!glyph.length) return 0;
    NSRect rect=[e.text.layoutManager boundingRectForGlyphRange:glyph inTextContainer:e.text.textContainer];
    rect=NSOffsetRect(rect,e.text.textContainerOrigin.x,e.text.textContainerOrigin.y);
    [e.text updateHoverAtPoint:NSMakePoint(NSMidX(rect),NSMidY(rect))];
    return 1;
}
char *cr_test_footnote_card(uintptr_t ptr) {
    NSData *json=[NSJSONSerialization dataWithJSONObject:[CRE(ptr).text footnoteCardState] options:0 error:NULL];
    NSString *value=json ? [[NSString alloc] initWithData:json encoding:NSUTF8StringEncoding] : nil;
    return strdup(value.UTF8String ?: "{}");
}
int cr_test_footnote_card_jump(uintptr_t ptr) {
    CREditor *e=CRE(ptr);
    if (![[e.text footnoteCardState][@"jump"] boolValue]) return 0;
    [e.text footnoteCardJump];
    return 1;
}
int cr_test_footnote_shot(uintptr_t ptr,const char *path) {
    return [CRE(ptr).text footnoteCardSnapshotTo:[NSString stringWithUTF8String:path]] ? 1 : 0;
}
int cr_test_emoji_shot(uintptr_t ptr,const char *path) {
    return [CRE(ptr) emojiMenuSnapshotTo:[NSString stringWithUTF8String:path]] ? 1 : 0;
}
int cr_test_link_shot(uintptr_t ptr,const char *path) {
    return [CRE(ptr) linkMenuSnapshotTo:[NSString stringWithUTF8String:path]] ? 1 : 0;
}
char *cr_test_page_cards(uintptr_t ptr) {
    NSData *json=[NSJSONSerialization dataWithJSONObject:[CRE(ptr) pageCardState] options:0 error:NULL];
    NSString *value=json ? [[NSString alloc] initWithData:json encoding:NSUTF8StringEncoding] : nil;
    return strdup(value.UTF8String ?: "{}");
}
char *cr_test_link(uintptr_t ptr) {
    NSData *json=[NSJSONSerialization dataWithJSONObject:[CRE(ptr) linkMenuState] options:0 error:NULL];
    NSString *value=json ? [[NSString alloc] initWithData:json encoding:NSUTF8StringEncoding] : nil;
    return strdup(value.UTF8String ?: "{}");
}
char *cr_test_emoji(uintptr_t ptr) {
    NSData *json=[NSJSONSerialization dataWithJSONObject:[CRE(ptr) emojiMenuState] options:0 error:NULL];
    NSString *value=json ? [[NSString alloc] initWithData:json encoding:NSUTF8StringEncoding] : nil;
    return strdup(value.UTF8String ?: "{}");
}
char *cr_test_hover(uintptr_t ptr) {
    NSData *json=[NSJSONSerialization dataWithJSONObject:[CRE(ptr).text hoverState] options:0 error:NULL];
    NSString *value=json ? [[NSString alloc] initWithData:json encoding:NSUTF8StringEncoding] : nil;
    return strdup(value.UTF8String ?: "{}");
}
int cr_test_hover_code(uintptr_t ptr,int activate) {
    CREditor *e=CRE(ptr);
    NSDictionary *block=[e codeBlocks].firstObject;
    if (!block) return 0;
    NSRect chip=[e.text codeChipRectForBlock:block];
    if (chip.size.height<=0) return 0;
    NSPoint point=NSMakePoint(NSMaxX(chip)-19*e.zoom,NSMinY(chip)+16*e.zoom);
    [e.text updateHoverAtPoint:point];
    if (activate) return [e.text copyHoveredCode] ? 1 : 0;
    return [[e.text hoverState][@"copy"] boolValue] ? 1 : 0;
}
int cr_test_click_code_text(uintptr_t ptr) {
    CREditor *e=CRE(ptr);
    NSDictionary *block=[e codeBlocks].firstObject;
    if (!block) return 0;
    NSRect chip=[e.text codeChipRectForBlock:block];
    if (chip.size.height<=0) return 0;
    NSPoint point=NSMakePoint(NSMinX(chip)+40*e.zoom,NSMidY(chip));
    NSPoint windowPoint=[e.text convertPoint:point toView:nil];
    NSEvent *down=[NSEvent mouseEventWithType:NSEventTypeLeftMouseDown location:windowPoint modifierFlags:0 timestamp:0 windowNumber:e.window.windowNumber context:nil eventNumber:0 clickCount:1 pressure:1];
    NSEvent *up=[NSEvent mouseEventWithType:NSEventTypeLeftMouseUp location:windowPoint modifierFlags:0 timestamp:0 windowNumber:e.window.windowNumber context:nil eventNumber:1 clickCount:1 pressure:0];
    [NSApp postEvent:up atStart:YES];
    [e.text mouseDown:down];
    NSRange range=[block[@"range"] rangeValue];
    NSUInteger caret=e.text.selectedRange.location;
    if (caret<range.location || caret>NSMaxRange(range)) return -1;
    return (int)caret;
}
int cr_test_drag_block(uintptr_t ptr,int fromParagraph,int toParagraph,int after) {
    CREditor *e=CRE(ptr);
    NSArray *ranges=CRRanges(e.text.string);
    if (fromParagraph<0 || toParagraph<0 || fromParagraph>=(NSInteger)ranges.count || toParagraph>=(NSInteger)ranges.count) return 0;
    NSRange from=[ranges[fromParagraph] rangeValue], to=[ranges[toParagraph] rangeValue];
    NSPoint handle=NSMakePoint(e.text.textContainerOrigin.x-15*e.zoom,NSMidY([e.text lineRectForCharacter:from.location]));
    [e.text updateHoverAtPoint:handle];
    NSRect targetRect=[e blockRectForRange:to];
    if (targetRect.size.height<=0) targetRect=[e.text lineRectForCharacter:to.location];
    NSPoint target=NSMakePoint(NSMidX(targetRect),after ? NSMaxY(targetRect)+2 : NSMinY(targetRect)-2);
    [e.text performTestDragTo:target];
    return 1;
}
int cr_test_hover_image(uintptr_t ptr,int buttonIndex,int activate) {
    CREditor *e=CRE(ptr);
    NSArray *figures=[e imageFigureRanges];
    if (!figures.count) return 0;
    NSRange range=[[figures firstObject] rangeValue];
    NSRange content=CRContent(e.text.string,range);
    NSRange glyph=[e.text.layoutManager glyphRangeForCharacterRange:content actualCharacterRange:NULL];
    if (!glyph.length) return 0;
    NSRect rect=[e.text.layoutManager boundingRectForGlyphRange:glyph inTextContainer:e.text.textContainer];
    rect=NSOffsetRect(rect,e.text.textContainerOrigin.x,e.text.textContainerOrigin.y);
    [e.text updateHoverAtPoint:NSMakePoint(NSMidX(rect),NSMidY(rect))];
    if (activate) return [e.text applyImageSizeAtIndex:buttonIndex] ? 1 : 0;
    return [[e.text hoverState][@"image"] boolValue] ? 1 : 0;
}
int cr_test_language_hover(uintptr_t ptr) {
    CREditor *e=CRE(ptr);
    NSDictionary *block=[e codeBlocks].firstObject;
    if (!block) return 0;
    NSRect chip=[e.text codeChipRectForBlock:block];
    if (chip.size.height<=0) return 0;
    [e.text updateHoverAtPoint:NSMakePoint(NSMaxX(chip)-19*e.zoom,NSMinY(chip)+16*e.zoom)];
    return [[e.text hoverState][@"language"] boolValue] ? 1 : 0;
}
int cr_test_language_pick(uintptr_t ptr,const char *language) {
    CREditor *e=CRE(ptr);
    NSDictionary *block=[e codeBlocks].firstObject;
    if (!block) return 0;
    NSString *name=[NSString stringWithUTF8String:language];
    NSMenuItem *item=[[NSMenuItem alloc] initWithTitle:name action:NULL keyEquivalent:@""];
    item.representedObject=@{@"language": name, @"range": block[@"range"]};
    [e.text pickCodeLanguage:item];
    return 1;
}
int cr_test_drag_preview(uintptr_t ptr,int fromParagraph,int toParagraph,int after) {
    CREditor *e=CRE(ptr);
    NSArray *ranges=CRRanges(e.text.string);
    if (fromParagraph<0 || toParagraph<0 || fromParagraph>=(NSInteger)ranges.count || toParagraph>=(NSInteger)ranges.count) return 0;
    NSRange from=[ranges[fromParagraph] rangeValue], to=[ranges[toParagraph] rangeValue];
    NSPoint handle=NSMakePoint(e.text.textContainerOrigin.x-15*e.zoom,NSMidY([e.text lineRectForCharacter:from.location]));
    [e.text updateHoverAtPoint:handle];
    NSRect targetRect=[e blockRectForRange:to];
    if (targetRect.size.height<=0) targetRect=[e.text lineRectForCharacter:to.location];
    NSPoint target=NSMakePoint(NSMidX(targetRect),after ? NSMaxY(targetRect)+2 : NSMinY(targetRect)-2);
    [e.text prepareTestDragTo:target];
    return 1;
}
int cr_test_hover_block(uintptr_t ptr,int paragraph,int activate) {
    CREditor *e=CRE(ptr);
    NSArray *ranges=CRRanges(e.text.string);
    if (paragraph<0 || paragraph>=(NSInteger)ranges.count) return 0;
    NSRange range=[ranges[paragraph] rangeValue];
    if (!range.length) return 0;
    NSPoint point=NSMakePoint(e.text.textContainerOrigin.x-15*e.zoom,NSMidY([e.text lineRectForCharacter:range.location]));
    [e.text updateHoverAtPoint:point];
    if (activate) return [e.text prepareBlockAtPoint:point] ? 1 : 0;
    return [[e.text hoverState][@"handle"] boolValue] ? 1 : 0;
}
int cr_test_format_bar_shot(uintptr_t ptr,const char *path) {
    return [CRE(ptr) formatBarSnapshotTo:[NSString stringWithUTF8String:path]] ? 1 : 0;
}
void cr_test_action(uintptr_t ptr,const char *action,const char *text) {
    CREditor *e=CRE(ptr); NSString *a=[NSString stringWithUTF8String:action], *s=[NSString stringWithUTF8String:text];
    if ([a isEqual:@"undo"]) { [e.text breakUndoCoalescing]; [e.undo undo]; }
    else if ([a isEqual:@"toc"]) {
        // Pick one entry of the first [TOC] figure, the way a click would.
        NSValue *value=[e tocFigureRanges].firstObject;
        if (!value) return;
        NSUInteger glyph=[e.text.layoutManager glyphIndexForCharacterAtIndex:value.rangeValue.location];
        NSRect rect=[e.text.layoutManager boundingRectForGlyphRange:NSMakeRange(glyph,1) inTextContainer:e.text.textContainer];
        NSPoint point=NSMakePoint(NSMidX(rect)+e.text.textContainerOrigin.x,rect.origin.y+e.text.textContainerOrigin.y+([s integerValue]+0.5)*CRTOCLine(e.zoom));
        NSEvent *event=[NSEvent mouseEventWithType:NSEventTypeLeftMouseDown location:[e.text convertPoint:point toView:nil] modifierFlags:0 timestamp:0 windowNumber:e.window.windowNumber context:nil eventNumber:0 clickCount:1 pressure:1];
        [e.text mouseDown:event];
    }
    else if ([a isEqual:@"key"]) {
        NSString *characters=[s hasSuffix:@"return"] ? @"\r" : [s hasSuffix:@"backspace"] ? @"\x7f" : [s hasSuffix:@"left"] ? @"[" : @"]";
        NSEventModifierFlags flags=[s hasPrefix:@"alt-"] ? NSEventModifierFlagOption : NSEventModifierFlagCommand;
        // Named structural shortcuts send an unmodified key with the right flags.
        if ([s isEqual:@"cmd-shift-up"] || [s isEqual:@"cmd-shift-down"]) {
            characters=[s hasSuffix:@"up"] ? @"\uf700" : @"\uf701";
            flags=NSEventModifierFlagCommand|NSEventModifierFlagShift;
        } else if ([s isEqual:@"cmd-d"]) {
            characters=@"d"; flags=NSEventModifierFlagCommand;
        } else if ([s isEqual:@"cmd-shift-backspace"]) {
            characters=@"\x7f"; flags=NSEventModifierFlagCommand|NSEventModifierFlagShift;
        }
        if ([s isEqual:@"footnote-back"]) { characters=@"\uf702"; flags=NSEventModifierFlagCommand|NSEventModifierFlagOption; }
        else if ([s hasPrefix:@"footnote-"]) { characters=[s hasSuffix:@"down"] ? @"\uf701" : @"\uf700"; flags=NSEventModifierFlagCommand|NSEventModifierFlagOption; }
        if ([s containsString:@"shift"]) flags |= NSEventModifierFlagShift;
        // plain-* sends an unmodified key, the way a writer presses it.
        if ([s hasPrefix:@"plain-"]) {
            NSString *name=[s substringFromIndex:6];
            characters=[name isEqual:@"return"] ? @"\r" : [name isEqual:@"escape"] ? @"\x1b" : [name isEqual:@"up"] ? @"\uf700" : [name isEqual:@"down"] ? @"\uf701" : [name isEqual:@"backspace"] ? @"\x7f" : name;
            flags=0;
        }
        NSEvent *event=[NSEvent keyEventWithType:NSEventTypeKeyDown location:NSZeroPoint modifierFlags:flags timestamp:0 windowNumber:e.window.windowNumber context:nil characters:characters charactersIgnoringModifiers:characters isARepeat:NO keyCode:0];
        [e.text keyDown:event];
    }
    else if ([a isEqual:@"slash-pick"]) [e slashChooseIndex:[s integerValue]];
    else if ([a isEqual:@"slash-close"]) [e slashClose];
    else if ([a isEqual:@"emoji-pick"]) [e emojiMenuChooseIndex:[s integerValue]];
    else if ([a isEqual:@"link-pick"]) [e linkMenuChooseIndex:[s integerValue]];
    else if ([a isEqual:@"formatbar-pick"]) [e formatBarApply:s];
    else if ([a isEqual:@"formatbar-link"]) [e formatBarLink:s];
    else if ([a isEqual:@"context"] || [a isEqual:@"context-at"] || [a isEqual:@"click"] || [a isEqual:@"click-trailing"] || [a isEqual:@"drag"]) {
        [e.text.layoutManager ensureLayoutForTextContainer:e.text.textContainer];
        NSUInteger target=e.text.selectedRange.location;
        if ([a isEqual:@"context-at"]) {
            NSArray *parts=[s componentsSeparatedByString:@"|"];
            if (parts.count!=2) return;
            target=MAX(0,[parts[0] integerValue]); s=parts[1];
        }
        NSRange glyph=[e.text.layoutManager glyphRangeForCharacterRange:NSMakeRange(target,1) actualCharacterRange:NULL];
        NSRect rect=[e.text.layoutManager boundingRectForGlyphRange:glyph inTextContainer:e.text.textContainer];
        NSPoint point=NSMakePoint(NSMinX(rect)+e.text.textContainerOrigin.x+1,NSMidY(rect)+e.text.textContainerOrigin.y);
        if ([a isEqual:@"click-trailing"]) point.x=NSMaxX(rect)+e.text.textContainerOrigin.x-1;
        point=[e.text convertPoint:point toView:nil];
        BOOL click=![a isEqual:@"context"] && ![a isEqual:@"context-at"];
        NSEvent *event=[NSEvent mouseEventWithType:click ? NSEventTypeLeftMouseDown : NSEventTypeRightMouseDown location:point modifierFlags:0 timestamp:0 windowNumber:e.window.windowNumber context:nil eventNumber:0 clickCount:1 pressure:1];
        if (click) {
            NSEvent *up=[NSEvent mouseEventWithType:NSEventTypeLeftMouseUp location:point modifierFlags:0 timestamp:0 windowNumber:e.window.windowNumber context:nil eventNumber:1 clickCount:1 pressure:0];
            [NSApp postEvent:up atStart:YES];
            if ([a isEqual:@"drag"]) {
                NSEvent *drag=[NSEvent mouseEventWithType:NSEventTypeLeftMouseDragged location:NSMakePoint(point.x+24,point.y) modifierFlags:0 timestamp:0 windowNumber:e.window.windowNumber context:nil eventNumber:2 clickCount:1 pressure:1];
                [NSApp postEvent:drag atStart:YES];
            }
            [e.text mouseDown:event]; return;
        }
        NSMenu *menu=[e.text menuForEvent:event];
        for (NSInteger i=0; i<menu.numberOfItems; i++) {
            id represented=[menu itemAtIndex:i].representedObject;
            NSString *command=[represented isKindOfClass:NSDictionary.class] ? represented[@"command"] : represented;
            if ([command isEqual:s]) { [menu performActionForItemAtIndex:i]; break; }
        }
    }
    else if ([a isEqual:@"newline"]) [e.text insertNewline:nil];
    else if ([a isEqual:@"type"]) [e.text insertText:s replacementRange:e.text.selectedRange];
    else if ([a isEqual:@"reveal"]) { if (![e revealInlineAt:e.text.selectedRange.location]) [e revealSourceAt:e.text.selectedRange.location]; }
    else if ([a isEqual:@"move-right"]) [e.text moveRight:nil];
    else if ([a isEqual:@"move-left"]) [e.text moveLeft:nil];
    else if ([a isEqual:@"click-below"]) {
        // A click in the blank page under the last line.
        [e.text.layoutManager ensureLayoutForTextContainer:e.text.textContainer];
        NSRect used=[e.text.layoutManager usedRectForTextContainer:e.text.textContainer];
        NSPoint point=[e.text convertPoint:NSMakePoint(e.text.textContainerOrigin.x+20,e.text.textContainerOrigin.y+NSMaxY(used)+40) toView:nil];
        NSEvent *down=[NSEvent mouseEventWithType:NSEventTypeLeftMouseDown location:point modifierFlags:0 timestamp:0 windowNumber:e.window.windowNumber context:nil eventNumber:0 clickCount:1 pressure:1];
        NSEvent *up=[NSEvent mouseEventWithType:NSEventTypeLeftMouseUp location:point modifierFlags:0 timestamp:0 windowNumber:e.window.windowNumber context:nil eventNumber:1 clickCount:1 pressure:0];
        [NSApp postEvent:up atStart:YES];
        [e.text mouseDown:down];
    }
    else if ([a isEqual:@"anchor"]) { if ([s hasPrefix:@"#"]) [e jumpToAnchor:[s substringFromIndex:1]]; } // what Cmd-click does
    else if ([a isEqual:@"layout"]) [e.text.layoutManager ensureLayoutForTextContainer:e.text.textContainer];
    else if ([a isEqual:@"display"]) {
        // Render the visible page the way a frame would, without layout help.
        [e.text scrollRangeToVisible:e.text.selectedRange];
        NSRect visible=e.text.visibleRect;
        NSBitmapImageRep *rep=[e.text bitmapImageRepForCachingDisplayInRect:visible];
        [e.text cacheDisplayInRect:visible toBitmapImageRep:rep];
    }
    else if ([a isEqual:@"move-down"]) [e.text moveDown:nil];
    else if ([a isEqual:@"move-up"]) [e.text moveUp:nil];
    else if ([a isEqual:@"blur"]) [e textDidEndEditing:[NSNotification notificationWithName:NSTextDidEndEditingNotification object:e.text]];
    else if ([a isEqual:@"linebreak"]) [e.text insertLineBreak:nil];
    else if ([a isEqual:@"tab"]) [e.text insertTab:nil];
    else if ([a isEqual:@"backtab"]) [e.text insertBacktab:nil];
    else if ([a isEqual:@"redo"]) [e.undo redo];
    else if ([a isEqual:@"paste"]) { [e.text breakUndoCoalescing]; [NSPasteboard.generalPasteboard clearContents]; [NSPasteboard.generalPasteboard setString:s forType:NSPasteboardTypeString]; [e.text paste:nil]; [e.text breakUndoCoalescing]; }
    else if ([a isEqual:@"plainPaste"]) { [NSPasteboard.generalPasteboard clearContents]; [NSPasteboard.generalPasteboard setString:s forType:NSPasteboardTypeString]; [e.text pasteAsPlainText:nil]; }
    else if ([a isEqual:@"copy"]) [e.text copy:nil];
    else if ([a isEqual:@"cut"]) [e.text cut:nil];
    else if ([a isEqual:@"pasteboard"]) [e.text paste:nil];
    else if ([a isEqual:@"underlinePaste"]) { NSMutableAttributedString *rich=[[NSMutableAttributedString alloc] initWithString:@"underlined" attributes:@{NSUnderlineStyleAttributeName:@1}]; NSData *rtf=[rich RTFFromRange:NSMakeRange(0,rich.length) documentAttributes:@{}]; [NSPasteboard.generalPasteboard clearContents]; [NSPasteboard.generalPasteboard setData:rtf forType:NSPasteboardTypeRTF]; [e.text paste:nil]; }
    else if ([a isEqual:@"unsupportedPaste"]) { NSMutableAttributedString *rich=[[NSMutableAttributedString alloc] initWithString:@"superscript" attributes:@{NSSuperscriptAttributeName:@1}]; NSData *rtf=[rich RTFFromRange:NSMakeRange(0,rich.length) documentAttributes:@{}]; [NSPasteboard.generalPasteboard clearContents]; [NSPasteboard.generalPasteboard setData:rtf forType:NSPasteboardTypeRTF]; [e.text paste:nil]; }
    else if ([a isEqual:@"backspace"]) [e.text deleteBackward:nil];
    else if ([a isEqual:@"delete"]) [e.text deleteForward:nil];
    else if ([a isEqual:@"pngPaste"]) { NSBitmapImageRep *rep=[[NSBitmapImageRep alloc] initWithBitmapDataPlanes:NULL pixelsWide:4 pixelsHigh:4 bitsPerSample:8 samplesPerPixel:4 hasAlpha:YES isPlanar:NO colorSpaceName:NSCalibratedRGBColorSpace bytesPerRow:0 bitsPerPixel:0]; [NSPasteboard.generalPasteboard clearContents]; [NSPasteboard.generalPasteboard setData:[rep representationUsingType:NSBitmapImageFileTypePNG properties:@{}] forType:NSPasteboardTypePNG]; [e.text paste:nil]; }
    else if ([a isEqual:@"htmlPaste"]) { [NSPasteboard.generalPasteboard clearContents]; [NSPasteboard.generalPasteboard setString:s forType:NSPasteboardTypeHTML]; [e.text paste:nil]; }
    else if ([a isEqual:@"filePaste"]) { [NSPasteboard.generalPasteboard clearContents]; [NSPasteboard.generalPasteboard setPropertyList:[s componentsSeparatedByString:@"|"] forType:NSFilenamesPboardType]; [e.text paste:nil]; }
    else if ([a isEqual:@"mark"]) [e.text setMarkedText:s selectedRange:NSMakeRange(s.length,0) replacementRange:e.text.selectedRange];
    else if ([a isEqual:@"commit"]) [e.text insertText:s replacementRange:e.text.markedRange];
    // Real IMEs commit with NSNotFound, letting the view replace marked text.
    else if ([a isEqual:@"commitIME"]) [e.text insertText:s replacementRange:NSMakeRange(NSNotFound,0)];
    else [e.text insertText:s replacementRange:e.text.selectedRange];
}
int cr_test_traits(uintptr_t ptr,const char *trait) { CREditor *e=CRE(ptr); NSString *key=[NSString stringWithUTF8String:trait]; __block BOOL found=NO; [e.text.textStorage enumerateAttributesInRange:NSMakeRange(0,e.text.string.length) options:0 usingBlock:^(NSDictionary *attrs, NSRange r, BOOL *stop) { if ([key isEqual:@"bold"]) found |= ([[NSFontManager sharedFontManager] traitsOfFont:attrs[NSFontAttributeName]] & NSBoldFontMask)!=0; else if ([key isEqual:@"italic"]) found |= [attrs[NSObliquenessAttributeName] doubleValue]>0; else if ([key isEqual:@"underline"]) found |= [attrs[NSUnderlineStyleAttributeName] integerValue]!=0; else if ([key isEqual:@"highlight"]) found |= [attrs[CRHighlight] boolValue]; else if ([key isEqual:@"sup"]) found |= [attrs[NSSuperscriptAttributeName] integerValue]>0; else if ([key isEqual:@"sub"]) found |= [attrs[NSSuperscriptAttributeName] integerValue]<0; else if ([key hasPrefix:@"hl:"]) found |= [attrs[CRHl] isEqual:[key substringFromIndex:3]]; else found |= CRKind(attrs[CRParagraph],key); }];
    if ([key isEqual:@"scrolled"]) return e.text.visibleRect.origin.y > 0.5;
    if ([key isEqual:@"attachments-clean"]) {
        __block BOOL clean=YES;
        [e.text.textStorage enumerateAttribute:NSAttachmentAttributeName inRange:NSMakeRange(0,e.text.string.length) options:0 usingBlock:^(id attachment, NSRange range, BOOL *stop) {
            if (attachment && (range.length!=1 || [e.text.string characterAtIndex:range.location]!=0xFFFC)) clean=NO;
        }];
        return clean;
    }
    if ([key isEqual:@"inline-math"] || [key isEqual:@"inline-source"]) {
        __block BOOL present=NO;
        [e.text.textStorage enumerateAttributesInRange:NSMakeRange(0,e.text.string.length) options:0 usingBlock:^(NSDictionary *attrs, NSRange range, BOOL *stop) {
            present |= [key isEqual:@"inline-source"] ? attrs[CRInlineSource]!=nil : (attrs[CRMathExpr]!=nil && attrs[NSAttachmentAttributeName]!=nil);
        }];
        return present;
    }
    if ([key hasPrefix:@"math-number:"]) {
        __block BOOL numbered=NO;
        [e.text.textStorage enumerateAttributesInRange:NSMakeRange(0,e.text.textStorage.length) options:0 usingBlock:^(NSDictionary *attrs, NSRange range, BOOL *stop) {
            NSTextAttachment *attachment=attrs[NSAttachmentAttributeName];
            if ([attrs[CRMathNumber] isEqual:[key substringFromIndex:12]] && attachment.image.size.width>0 && attachment.image.size.height>0) numbered=YES;
        }];
        return numbered;
    }
    if ([key hasPrefix:@"alert:"]) {
        // Reports the callout label a paragraph carries, which lives only in
        // the paragraph attribute: the marker itself stays ordinary text.
        NSString *want=[key substringFromIndex:6];
        __block BOOL present=NO;
        [e.text.textStorage enumerateAttribute:CRParagraph inRange:NSMakeRange(0,e.text.textStorage.length) options:0 usingBlock:^(id value, NSRange range, BOOL *stop) {
            if (![value isKindOfClass:NSDictionary.class]) return;
            NSString *have=value[@"alert"];
            if (![have isKindOfClass:NSString.class]) have=@"";
            if ([have isEqual:want]) present=YES;
        }];
        return present;
    }
    if ([key isEqual:@"findbar"]) return e.text.usesFindBar ? 1 : 0;
    if ([key isEqual:@"replacebar"]) {
        // NSTextFinderActionShowReplaceInterface (12) is only meaningful when
        // the text view accepts it; validation is what the menu would ask.
        NSMenuItem *item=[NSMenuItem new];
        item.action=@selector(performFindPanelAction:);
        item.tag=12;
        return [e.text validateUserInterfaceItem:item] && e.text.usesFindBar ? 1 : 0;
    }
    if ([key hasPrefix:@"alphaAt:"]) { NSInteger index=[[key substringFromIndex:8] integerValue]; if (index>=0 && index<e.text.textStorage.length) { NSColor *color=[e.text.textStorage attribute:NSForegroundColorAttributeName atIndex:index effectiveRange:NULL]; return color && color.alphaComponent<0.99; } return 0; }
    return found; }
int cr_test_checkbox(uintptr_t ptr) { CREditor *e=CRE(ptr); for (NSValue *v in CRRanges(e.text.string)) if (CRKind(CRParagraphAt(e,v.rangeValue.location),@"checklist")) { NSRect r=[e markerRect:v.rangeValue]; NSPoint point=[e.text convertPoint:NSMakePoint(NSMidX(r),NSMidY(r)) toView:nil]; NSEvent *event=[NSEvent mouseEventWithType:NSEventTypeLeftMouseDown location:point modifierFlags:0 timestamp:0 windowNumber:e.window.windowNumber context:nil eventNumber:0 clickCount:1 pressure:1]; [e.text mouseDown:event]; return 1; } return 0; }
void cr_test_opaque(uintptr_t ptr,int whole) { CREditor *e=CRE(ptr); for (NSValue *v in CRRanges(e.text.string)) if (CRKind(CRParagraphAt(e,v.rangeValue.location),@"opaque")) { NSRange r=CRContent(e.text.string,v.rangeValue); if (!whole) r=NSMakeRange(r.location+1,1); e.text.selectedRange=r; return; } }
int cr_test_screenshot(uintptr_t ptr,const char *path) { CREditor *e=CRE(ptr); [e.scroll layoutSubtreeIfNeeded]; [e.text display]; [e.text.layoutManager ensureLayoutForTextContainer:e.text.textContainer]; NSBitmapImageRep *rep=[e.scroll bitmapImageRepForCachingDisplayInRect:e.scroll.bounds]; [e.scroll cacheDisplayInRect:e.scroll.bounds toBitmapImageRep:rep]; NSData *png=[rep representationUsingType:NSBitmapImageFileTypePNG properties:@{}]; return [png writeToFile:[NSString stringWithUTF8String:path] atomically:YES]; }
@interface CRFlippedTestView : NSView
@end
@implementation CRFlippedTestView
- (BOOL)isFlipped { return YES; }
- (BOOL)acceptsFirstResponder { return YES; }
@end

int cr_test_layout(uintptr_t ptr) {
    CREditor *e=CRE(ptr);
    if (!e.scroll.hidden) return 0;
    e.window.contentView.wantsLayer=YES;
    cr_frame(ptr,11,22,300,200);
    if (!NSEqualRects(e.scroll.frame,NSMakeRect(11,278,300,200))) return 0;
    CRFlippedTestView *flipped=[[CRFlippedTestView alloc] initWithFrame:e.window.contentView.bounds];
    flipped.wantsLayer=YES; [e.scroll removeFromSuperview]; [e.window.contentView addSubview:flipped]; [flipped addSubview:e.scroll positioned:NSWindowAbove relativeTo:nil];
    cr_frame(ptr,11,22,300,200);
    if (!NSEqualRects(e.scroll.frame,NSMakeRect(11,22,300,200))) return 0;
    [e.window makeFirstResponder:flipped]; cr_visible(ptr,1); cr_focus(ptr,1);
    if (e.window.firstResponder!=e.text || e.scroll.hidden) return 0;
    cr_focus(ptr,0); if (e.window.firstResponder!=flipped) return 0;
    cr_focus(ptr,1); cr_visible(ptr,0);
    if (!e.scroll.hidden || e.window.firstResponder!=flipped) return 0;
    e.previousResponder=nil; cr_visible(ptr,1);
    [e.window makeFirstResponder:e.text]; // AppKit mouse focus, not Editor.Focus.
    cr_focus(ptr,0);
    return e.window.firstResponder==flipped;
}


// ===========================================================================
// Typora-parity additions
// ===========================================================================

@implementation CREditor (Parity)

// emojiRule replaces a ":word:" shortcode right before the caret with its
// emoji. boundary=YES means the just-typed space/tab sits in the storage
// after the shortcode; boundary=NO is the Enter that newline: is handling.
- (BOOL)emojiRule:(BOOL)boundary {
    // No _suppress guard: applyInputRules runs while insertText: holds suppress.
    if (!_text.editable || _text.hasMarkedText || _text.selectedRange.length) return NO;
    NSUInteger caret=_text.selectedRange.location;
    NSRange paragraphRange=[_text.string paragraphRangeForRange:NSMakeRange(caret,0)];
    NSDictionary *paragraph=CRParagraphAt(self,paragraphRange.location);
    if (CRKind(paragraph,@"opaque") || CRKind(paragraph,@"source") || CRKind(paragraph,@"code") || CRKind(paragraph,@"math")) return NO;
    if (_text.typingAttributes[CRCode] || _text.typingAttributes[CRMathText] || _text.typingAttributes[CRInlineSource]) return NO;
    if (caret && caret<=_text.textStorage.length) {
        NSDictionary *before=[_text.textStorage attributesAtIndex:caret-1 effectiveRange:NULL];
        if (before[CRCode] || before[CRMathText] || before[CRInlineSource]) return NO;
    }
    NSString *prefix=[_text.string substringWithRange:NSMakeRange(paragraphRange.location,caret-paragraphRange.location)];
    NSString *pattern=boundary ? @":([a-z0-9_+-]{2,}):[ \\t]$" : @":([a-z0-9_+-]{2,}):$";
    NSTextCheckingResult *match=[[NSRegularExpression regularExpressionWithPattern:pattern options:0 error:NULL] firstMatchInString:prefix options:0 range:NSMakeRange(0,prefix.length)];
    if (!match) return NO;
    NSString *word=[prefix substringWithRange:[match rangeAtIndex:1]];
    char *emoji=cr_emoji((char *)word.UTF8String);
    if (!emoji) return NO;
    NSString *value=[NSString stringWithUTF8String:emoji]; free(emoji);
    if (!value.length) return NO;
    NSRange range=NSMakeRange(paragraphRange.location+match.range.location,match.range.length-(boundary ? 1 : 0));
    NSDictionary *attrs=[_text.textStorage attributesAtIndex:range.location effectiveRange:NULL];
    NSAttributedString *replacement=[[NSAttributedString alloc] initWithString:value attributes:attrs];
    [self replace:range with:replacement selection:NSMakeRange(range.location+value.length+(boundary ? 1 : 0),0)];
    return YES;
}

// autoPair gives editor-style bracket/quote pairing: an opener inserts the
// pair with the caret inside, a closer steps over an existing one
// (type-over), and a pair character with a selection wraps it. Never inside
// code, source, or math.
// CRTextAttrs drops formula, footnote and image identity from attributes
// borrowed for typed text: only the attachment character carries those.
static NSDictionary *CRTextAttrs(NSDictionary *attrs) {
    if (!attrs[NSAttachmentAttributeName]) return attrs;
    NSMutableDictionary *plain=[attrs mutableCopy];
    for (NSString *key in @[CRMathText,CRMathExpr,CRMathNumber,CRMathRendered,CRFootnote,CRFootnoteNumber,CRImageSource,CRImageURL,NSToolTipAttributeName,NSAttachmentAttributeName]) [plain removeObjectForKey:key];
    return plain;
}
- (BOOL)autoPair:(id)value range:(NSRange)range {
    NSString *s=[value isKindOfClass:NSAttributedString.class] ? [value string] : value;
    if (_suppress || !_text.editable || _text.hasMarkedText) return NO;
    if (s.length!=1) return NO;
    unichar c=[s characterAtIndex:0];
    NSString *closer=CRPairCloser(s);
    BOOL wrapOnly=!closer && [@"`*_~" rangeOfString:s].location!=NSNotFound;
    BOOL isCloser=[@")]}\"'）】》”’」』" rangeOfString:s].location!=NSNotFound;
    if (!closer && !wrapOnly && !isCloser) return NO;
    NSDictionary *p=CRParagraphAt(self,range.location);
    if (CRKind(p,@"code") || CRKind(p,@"source") || CRKind(p,@"math") || CRKind(p,@"opaque")) return NO;
    if (CRCellAt(self,range.location)) return NO;
    if (_text.typingAttributes[CRCode] || _text.typingAttributes[CRMathText] || _text.typingAttributes[CRInlineSource]) return NO;
    if (range.location && range.location<=_text.textStorage.length) {
        NSDictionary *before=[_text.textStorage attributesAtIndex:range.location-1 effectiveRange:NULL];
        if (before[CRCode] || before[CRMathText] || before[CRInlineSource]) return NO;
    }
    if (range.length) {
        if (!closer && !wrapOnly) return NO;
        __block BOOL skip=NO;
        [_text.textStorage enumerateAttributesInRange:range options:0 usingBlock:^(NSDictionary *attrs,NSRange r,BOOL *stop) { if (attrs[CRCode] || attrs[CRMathText]) { skip=YES; *stop=YES; } }];
        if (skip) return NO;
        // Markdown markers format the selection (Typora): * italic, _ italic,
        // ~ strikethrough, ` code. Brackets and quotes wrap it literally.
        NSString *command=@{@"*":@"italic",@"_":@"italic",@"~":@"strike",@"`":@"code"}[s];
        if (command) { [self format:command]; return YES; }
        // Only the selection changes; replacing the whole note would relayout it.
        NSMutableAttributedString *wrapped=[[_text.textStorage attributedSubstringFromRange:range] mutableCopy];
        NSDictionary *openAttrs=CRTextAttrs([wrapped attributesAtIndex:0 effectiveRange:NULL]);
        NSDictionary *closeAttrs=CRTextAttrs([wrapped attributesAtIndex:wrapped.length-1 effectiveRange:NULL]);
        [wrapped appendAttributedString:[[NSAttributedString alloc] initWithString:closer ?: s attributes:closeAttrs]];
        [wrapped insertAttributedString:[[NSAttributedString alloc] initWithString:s attributes:openAttrs] atIndex:0];
        [self replace:range with:wrapped selection:NSMakeRange(range.location+1,range.length)];
        return YES;
    }
    NSString *string=_text.string;
    if (isCloser && range.location<string.length && [string characterAtIndex:range.location]==c) {
        _text.selectedRange=NSMakeRange(range.location+1,0);
        [self applyInputRules]; // stepping past a closer can complete a rule (a link's ")")
        return YES;
    }
    if (!closer) return NO;
    // A quote right after a word is a closing quote: keep it single.
    if ((c=='"' || c=='\'') && range.location && [NSCharacterSet.alphanumericCharacterSet characterIsMember:[string characterAtIndex:range.location-1]]) return NO;
    [self replace:range with:[[NSAttributedString alloc] initWithString:[s stringByAppendingString:closer] attributes:_text.typingAttributes] selection:NSMakeRange(range.location+1,0)];
    return YES;
}

// insertMarkdownText parses Markdown and inserts it at the caret through the
// same path cr_insert uses.
- (void)insertMarkdownText:(NSString *)markdown {
    if (!markdown.length) return;
    char *json=cr_parse_markdown((char *)markdown.UTF8String);
    if (!json) return;
    cr_insert((uintptr_t)(__bridge void *)self,json);
    free(json);
}

// applyFocusMode dims every paragraph but the caret's. It runs at the tail of
// restyle, so the colors it reads are always the undimmed ones.
- (void)applyFocusMode {
    NSArray *active=CRSelectedParagraphs(self);
    NSMutableArray *dim=[NSMutableArray array];
    for (NSValue *v in CRRanges(_text.string)) {
        NSRange r=v.rangeValue;
        BOOL isActive=NO;
        for (NSValue *a in active) if (NSEqualRanges(r,[a rangeValue])) { isActive=YES; break; }
        if (isActive || !r.length) continue;
        NSDictionary *p=CRParagraphAt(self,r.location);
        CGFloat alpha=(CRKind(p,@"code") || CRKind(p,@"source") || CRKind(p,@"opaque")) ? 0.5 : 0.35;
        [_text.textStorage enumerateAttribute:NSForegroundColorAttributeName inRange:r options:0 usingBlock:^(NSColor *color,NSRange range,BOOL *stop) {
            if (color) [dim addObject:@{@"range":[NSValue valueWithRange:range],@"color":[color colorWithAlphaComponent:alpha]}];
        }];
    }
    for (NSDictionary *d in dim) [_text.textStorage addAttribute:NSForegroundColorAttributeName value:d[@"color"] range:[d[@"range"] rangeValue]];
}
- (void)setFocusMode:(BOOL)enabled { if (_focusMode==enabled) return; _focusMode=enabled; [self restyle]; }

// centerCaretLine scrolls so the caret's line sits at the vertical center of
// the viewport; only when the caret moved and the document overflows.
- (void)centerCaretLine {
    if (!_typewriter || _text.hasMarkedText || _text.selectedRange.length) return;
    NSUInteger caret=_text.selectedRange.location;
    if (caret==_lastCaret) return;
    _lastCaret=caret;
    [_text.layoutManager ensureLayoutForTextContainer:_text.textContainer];
    NSRect visible=_text.visibleRect;
    if (!visible.size.height) return;
    CGFloat content=[_text.layoutManager usedRectForTextContainer:_text.textContainer].size.height+2*_text.textContainerInset.height;
    content=MAX(content,_text.frame.size.height);
    if (content<=visible.size.height) return;
    NSUInteger glyph=[_text.layoutManager glyphIndexForCharacterAtIndex:caret];
    NSRect line=glyph<_text.layoutManager.numberOfGlyphs ? [_text.layoutManager lineFragmentRectForGlyphAtIndex:glyph effectiveRange:NULL] : _text.layoutManager.extraLineFragmentRect;
    line=NSOffsetRect(line,_text.textContainerOrigin.x,_text.textContainerOrigin.y);
    CGFloat target=MAX(0,MIN(NSMidY(line)-visible.size.height/2,content-visible.size.height));
    NSClipView *clip=_scroll.contentView;
    [clip setBoundsOrigin:NSMakePoint(clip.bounds.origin.x,target)];
    [_scroll reflectScrolledClipView:clip];
}
- (void)setTypewriter:(BOOL)enabled { _typewriter=enabled; _lastCaret=NSNotFound; if (enabled) [self centerCaretLine]; }

- (void)showFindBar {
    _text.usesFindBar=YES; // incremental find bar inside the scroll view
    [self.window makeFirstResponder:_text];
    NSMenuItem *item=[NSMenuItem new];
    item.tag=NSFindPanelActionShowFindPanel;
    [_text performFindPanelAction:item];
}
// showReplaceBar opens the same bar straight in its replace mode, so the
// writer does not have to discover the mode switch inside the bar.
- (void)showReplaceBar {
    _text.usesFindBar=YES;
    [self.window makeFirstResponder:_text];
    NSMenuItem *item=[NSMenuItem new];
    item.tag=12; // NSTextFinderActionShowReplaceInterface
    [_text performFindPanelAction:item];
}
- (void)revealHeading:(NSInteger)index {
    NSInteger seen=0;
    for (NSValue *v in CRRanges(_text.string)) {
        NSRange r=v.rangeValue;
        if (!CRKind(CRParagraphAt(self,r.location),@"heading")) continue;
        if (seen++==index) { [_text scrollRangeToVisible:r]; return; }
    }
}
- (void)printDocument {
    if (!self.window) return;
    NSDictionary *doc=[self document];
    // Print is light-on-paper even in dark mode: rebuild display attributes.
    BOOL wasDark=_dark; _dark=NO;
    NSDictionary *ending=_emptyParagraph;
    NSAttributedString *content=[self attributedDocument:doc];
    _emptyParagraph=ending; _dark=wasDark;
    NSPrintInfo *info=[[NSPrintInfo sharedPrintInfo] copy];
    info.paperSize=NSMakeSize(595,842); // A4
    CGFloat width=info.imageablePageBounds.size.width;
    NSTextView *view=[[NSTextView alloc] initWithFrame:NSMakeRect(0,0,width,100)];
    view.textContainer.containerSize=NSMakeSize(width,CGFLOAT_MAX);
    view.textContainer.widthTracksTextView=YES;
    view.verticallyResizable=YES; view.horizontallyResizable=NO;
    view.maxSize=NSMakeSize(width,CGFLOAT_MAX);
    view.editable=NO;
    [view.textStorage setAttributedString:content];
    [view.layoutManager ensureLayoutForTextContainer:view.textContainer];
    NSRect used=[view.layoutManager usedRectForTextContainer:view.textContainer];
    [view setFrameSize:NSMakeSize(width,used.size.height+2*view.textContainerInset.height)];
    [[NSPrintOperation printOperationWithView:view printInfo:info] runOperationModalForWindow:self.window delegate:nil didRunSelector:NULL contextInfo:NULL];
}

@end

// Spellcheck never fights IME marked text: textDidChange and
// updateTypingParagraph skip their work while hasMarkedText is set, and the
// checker itself only reads committed text.
void cr_spellcheck(uintptr_t ptr, int enabled) { CRE(ptr).text.continuousSpellCheckingEnabled = enabled; }
void cr_focus_mode(uintptr_t ptr, int enabled) { [CRE(ptr) setFocusMode:enabled]; }
void cr_typewriter(uintptr_t ptr, int enabled) { [CRE(ptr) setTypewriter:enabled]; }
void cr_find(uintptr_t ptr) { [CRE(ptr) showFindBar]; }
void cr_find_replace(uintptr_t ptr) { [CRE(ptr) showReplaceBar]; }
void cr_reveal_heading(uintptr_t ptr, int index) { [CRE(ptr) revealHeading:(NSInteger)index]; }
void cr_print(uintptr_t ptr) { [CRE(ptr) printDocument]; }
// cr_test_markers reports each list marker's vertical center as drawRect
// would compute it now, then the same after full layout: "lazy|full".
char *cr_test_markers(uintptr_t ptr) {
    CREditor *e=CRE(ptr); NSMutableArray *lazy=[NSMutableArray array], *full=[NSMutableArray array];
    for (NSValue *v in CRRanges(e.text.string)) if (CRList(CRParagraphAt(e,v.rangeValue.location))) [lazy addObject:[NSString stringWithFormat:@"%.1f",NSMidY([e markerLineRect:v.rangeValue])]];
    [e.text.layoutManager ensureLayoutForTextContainer:e.text.textContainer];
    for (NSValue *v in CRRanges(e.text.string)) if (CRList(CRParagraphAt(e,v.rangeValue.location))) [full addObject:[NSString stringWithFormat:@"%.1f",NSMidY([e markerLineRect:v.rangeValue])]];
    return strdup([NSString stringWithFormat:@"%@|%@",[lazy componentsJoinedByString:@","],[full componentsJoinedByString:@","]].UTF8String);
}

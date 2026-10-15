//go:build darwin && cgo

#import <AppKit/AppKit.h>
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
static NSString *const CRUnderline = @"CorraUnderline";
static NSString *const CRHighlight = @"CorraHighlight";
static NSString *const CRSup = @"CorraSup";
static NSString *const CRSub = @"CorraSub";
static NSString *const CRHl = @"CorraHl"; // syntax-highlight token class on code
static NSString *const CROwnPaste = @"app.corra.rich-document";
// Go-side handlers for parity features (cgo exports, main thread only).
extern char *cr_emoji(const char *word);
extern char *cr_paste_image(uintptr_t handle, const char *pngBase64, const char *name);
extern char *cr_drop_files(uintptr_t handle, const char *pathsJSON);
extern char *cr_html_to_markdown(const char *html);
@class CREditor;
@interface CRTextView : NSTextView
@property(nonatomic, weak) CREditor *owner;
@end
@interface CREditor : NSObject <NSTextViewDelegate>
@property NSScrollView *scroll;
@property CRTextView *text;
@property NSUndoManager *undo;
@property(nonatomic, weak) NSWindow *window;
@property(nonatomic, weak) NSResponder *previousResponder;
@property uintptr_t callback;
@property BOOL suppress;
@property BOOL dark;
@property double zoom;
@property NSDictionary *emptyParagraph;
@property NSString *baseDir;
@property NSMutableDictionary *mathCache;
@property NSMutableDictionary *imageCache;
@property NSMutableDictionary *imagePending;
@property NSMutableDictionary *imageAttachments;
@property NSRange draftGroup; // contiguous source/body paragraphs the caret is in
@property BOOL focusMode;
@property BOOL typewriter;
@property NSUInteger lastCaret;
- (void)restyle;
- (void)normalize;
- (void)notify;
- (void)source;
- (BOOL)allowRange:(NSRange)range;
- (void)format:(NSString *)command;
- (void)replace:(NSRange)range with:(NSAttributedString *)value selection:(NSRange)selection;
- (void)restore:(NSAttributedString *)value selection:(NSRange)selection typing:(NSDictionary *)typing ending:(NSDictionary *)ending;
- (NSDictionary *)document;
- (NSDictionary *)documentForRange:(NSRange)selection;
- (NSAttributedString *)attributedDocument:(NSDictionary *)document;
- (void)newline;
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
- (NSRect)imageBounds:(NSImage *)img;
@end
@interface CREditor (Parity)
- (void)setFocusMode:(BOOL)enabled;
- (void)setTypewriter:(BOOL)enabled;
- (void)showFindBar;
- (void)revealHeading:(NSInteger)index;
- (void)printDocument;
- (BOOL)emojiRule:(BOOL)boundary;
- (BOOL)autoPair:(id)value range:(NSRange)range;
- (void)insertMarkdownText:(NSString *)markdown;
- (void)applyFocusMode;
- (void)centerCaretLine;
@end

static NSDictionary *CRBody(void) { return @{@"kind": @"body", @"runs": @[]}; }
// CRBlockEnding: the paragraph's source separator holds a blank line, so it
// is the last line of its block.
static BOOL CRBlockEnding(NSDictionary *p) {
    NSString *separator = [p[@"separator"] isKindOfClass:NSString.class] ? p[@"separator"] : @"";
    return [[separator stringByReplacingOccurrencesOfString:@"\r" withString:@""] rangeOfString:@"\n\n"].location != NSNotFound;
}
static BOOL CRKind(NSDictionary *p, NSString *kind) { return [p[@"kind"] isEqual:kind]; }
static NSDictionary *CRFresh(NSDictionary *p) {
    NSMutableDictionary *fresh = [@{@"kind": p[@"kind"] ?: @"body", @"runs": @[]} mutableCopy];
    if (p[@"level"]) fresh[@"level"] = p[@"level"];
    if (CRKind(p,@"ordered")) fresh[@"ordinal"] = @([p[@"ordinal"] integerValue] + 1);
    return fresh;
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
static NSRange CRContent(NSString *string, NSRange range) {
    while (range.length && [[NSCharacterSet newlineCharacterSet] characterIsMember:[string characterAtIndex:NSMaxRange(range)-1]]) range.length--;
    return range;
}
static NSDictionary *CRParagraphAt(CREditor *e, NSUInteger location) {
    if (location < e.text.textStorage.length) return [e.text.textStorage attribute:CRParagraph atIndex:location effectiveRange:NULL] ?: CRBody();
    return e.emptyParagraph ?: e.text.typingAttributes[CRParagraph] ?: CRBody();
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
            @"comment":[NSColor colorWithRed:0.50 green:0.52 blue:0.55 alpha:1],
            @"operator":[NSColor colorWithRed:0.40 green:0.22 blue:0.56 alpha:1],
            @"type":[NSColor colorWithRed:0.19 green:0.42 blue:0.66 alpha:1],
            @"name":[NSColor colorWithRed:0.47 green:0.30 blue:0.07 alpha:1],
        };
        darkMap=@{
            @"keyword":[NSColor colorWithRed:0.86 green:0.56 blue:0.90 alpha:1],
            @"string":[NSColor colorWithRed:0.58 green:0.82 blue:0.52 alpha:1],
            @"number":[NSColor colorWithRed:0.60 green:0.76 blue:0.98 alpha:1],
            @"literal":[NSColor colorWithRed:0.60 green:0.76 blue:0.98 alpha:1],
            @"comment":[NSColor colorWithRed:0.56 green:0.59 blue:0.63 alpha:1],
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

@implementation CREditor
- (void)notify { if (!_suppress && _callback) cr_callback(_callback,0); }
- (void)source { if (!_suppress && _callback) cr_callback(_callback,1); }
- (NSUndoManager *)undoManagerForTextView:(NSTextView *)view { return _undo; }
- (NSDictionary *)visual:(NSDictionary *)attrs paragraph:(NSDictionary *)p {
    NSMutableDictionary *out = [attrs mutableCopy];
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
    if ([attrs[CRCode] boolValue] && !heading) size -= 1.5 * _zoom;
    NSFont *font = ([attrs[CRCode] boolValue] || source) ? [NSFont monospacedSystemFontOfSize:size weight:NSFontWeightRegular] : [NSFont systemFontOfSize:size weight:(heading && level > 2) ? NSFontWeightSemibold : NSFontWeightRegular];
    NSFontTraitMask traits = 0;
    if ([attrs[CRBold] boolValue] || (heading && level <= 2) || (cell && cell.startingRow == 0)) traits |= NSBoldFontMask;
    if ([attrs[CRItalic] boolValue]) traits |= NSItalicFontMask;
    if (traits) font = [[NSFontManager sharedFontManager] convertFont:font toHaveTrait:traits];
    // Sup/sub shift the baseline and shrink the glyph; scrub the display
    // attribute first so a removed semantic trait does not linger.
    BOOL sup = [attrs[CRSup] boolValue], sub = [attrs[CRSub] boolValue];
    if (sup || sub) {
        font = [NSFont fontWithDescriptor:font.fontDescriptor size:font.pointSize * 0.75];
        out[NSSuperscriptAttributeName] = sup ? @1 : @-1;
    } else [out removeObjectForKey:NSSuperscriptAttributeName];
    out[NSFontAttributeName] = font;
    out[NSForegroundColorAttributeName] = _dark ? [NSColor colorWithRed:0.906 green:0.914 blue:0.933 alpha:1] : [NSColor colorWithWhite:0.161 alpha:1];
    // Quotes step back: same size, quieter ink, so the bar and the tint carry
    // the "cited, not mine" reading instead of a smaller font.
    if (CRKind(p,@"quote")) out[NSForegroundColorAttributeName] = _dark ? [NSColor colorWithWhite:0.62 alpha:1] : [NSColor colorWithWhite:0.42 alpha:1];
    NSMutableParagraphStyle *style = [NSMutableParagraphStyle new];
    // ~1.55 line height keeps mixed CJK/Latin lines from rubbing shoulders.
    style.lineSpacing = 6 * _zoom;
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
        style.firstLineHeadIndent = 30 * _zoom; style.headIndent = 30 * _zoom;
    }
    out[NSParagraphStyleAttributeName] = style;
    [out removeObjectForKey:NSBackgroundColorAttributeName];
    [out removeObjectForKey:NSUnderlineStyleAttributeName];
    if ([attrs[CRUnderline] boolValue]) out[NSUnderlineStyleAttributeName] = @(NSUnderlineStyleSingle);
    // Opaque chips are drawn in drawRect: a glyph background would paint
    // through the paragraph gap and merge adjacent blocks into one slab.
    if ([attrs[CRCode] boolValue]) {
        out[NSBackgroundColorAttributeName] = _dark ? [NSColor colorWithWhite:0.23 alpha:1] : [NSColor colorWithWhite:0.94 alpha:1];
    }
    if ([attrs[CRHighlight] boolValue]) {
        out[NSBackgroundColorAttributeName] = _dark ? [NSColor colorWithRed:0.75 green:0.55 blue:0.10 alpha:0.45] : [NSColor colorWithRed:1 green:0.87 blue:0.42 alpha:1];
    }
    if (CRKind(p,@"opaque")) {
        out[NSFontAttributeName] = [NSFont monospacedSystemFontOfSize:12.5*_zoom weight:NSFontWeightRegular];
        out[NSForegroundColorAttributeName] = _dark ? [NSColor colorWithWhite:0.65 alpha:1] : [NSColor colorWithWhite:0.43 alpha:1];
    }
    if (attrs[NSLinkAttributeName]) out[NSForegroundColorAttributeName] = NSColor.linkColor;
    if (attrs[CRHl]) out[NSForegroundColorAttributeName] = CRHlColor(attrs[CRHl],_dark);
    return out;
}
// highlightCode tokenizes each fenced code block with chroma (via cr_highlight)
// and tags its characters with CRHl token classes, which -visual: maps to
// theme colors. Contiguous "code" paragraphs form one block so multi-line
// strings and comments tokenize correctly; the fence lives in metadata, not in
// the text, so the block's language comes from its first paragraph's "open".
- (void)highlightCode {
    NSArray *ranges=CRRanges(_text.string);
    // Clear any stale token colors first; a re-parse may have changed the code.
    [_text.textStorage removeAttribute:CRHl range:NSMakeRange(0,_text.string.length)];
    for (NSUInteger i=0;i<ranges.count;) {
        NSDictionary *p=CRParagraphAt(self,[ranges[i] rangeValue].location);
        if (!CRKind(p,@"code")) { i++; continue; }
        // Accumulate the contiguous run of code lines into one block.
        NSString *lang=@""; NSString *open=[p[@"open"] isKindOfClass:NSString.class] ? p[@"open"] : @"";
        if (open.length) lang=[[open stringByTrimmingCharactersInSet:[NSCharacterSet characterSetWithCharactersInString:@"`~ "]] lowercaseString];
        NSUInteger start=CRContent(_text.string,[ranges[i] rangeValue]).location;
        NSUInteger end=start; NSUInteger j=i;
        for (;j<ranges.count;j++) {
            NSDictionary *q=CRParagraphAt(self,[ranges[j] rangeValue].location);
            if (!CRKind(q,@"code")) break;
            end=NSMaxRange(CRContent(_text.string,[ranges[j] rangeValue]));
        }
        i=j;
        if (end<=start) continue;
        if (!lang.length) continue; // no language tag: leave the code untinted
        NSRange block=NSMakeRange(start,end-start);
        NSString *code=[_text.string substringWithRange:block];
        char *out=cr_highlight((char *)code.UTF8String,(char *)lang.UTF8String);
        if (!out) continue;
        NSArray *spans=[NSJSONSerialization JSONObjectWithData:[[NSString stringWithUTF8String:out] dataUsingEncoding:NSUTF8StringEncoding] options:0 error:NULL];
        free(out);
        NSUInteger cursor=start;
        for (NSDictionary *span in spans) {
            NSString *text=span[@"Text"]; NSString *cls=span[@"Class"];
            NSUInteger len=text.length;
            if (cursor+len>end) len=end>cursor ? end-cursor : 0;
            if (len && cls.length && ![cls isEqual:@"text"]) [_text.textStorage addAttribute:CRHl value:cls range:NSMakeRange(cursor,len)];
            cursor+=text.length;
            if (cursor>=end) break;
        }
    }
}
- (void)restyle {
    BOOL wasSuppress = _suppress; _suppress = YES;
    NSArray *selection = _text.selectedRanges;
    [self highlightCode];
    [_text.textStorage beginEditing];
    NSAttributedString *copy = [_text.textStorage copy];
    [copy enumerateAttributesInRange:NSMakeRange(0,copy.length) options:0 usingBlock:^(NSDictionary *attrs, NSRange range, BOOL *stop) {
        [self.text.textStorage setAttributes:[self visual:attrs paragraph:attrs[CRParagraph] ?: CRBody()] range:range];
    }];
    [_text.textStorage endEditing];
    _text.typingAttributes = [self visual:_text.typingAttributes paragraph:_text.typingAttributes[CRParagraph] ?: _emptyParagraph ?: CRBody()];
    _text.selectedRanges = selection;
    NSColor *background = _dark ? [NSColor colorWithRed:0.149 green:0.153 blue:0.176 alpha:1] : NSColor.whiteColor;
    _text.backgroundColor = background; _scroll.backgroundColor = background;
    _text.insertionPointColor = _dark ? NSColor.whiteColor : NSColor.blackColor;
    _scroll.appearance = [NSAppearance appearanceNamed:_dark ? NSAppearanceNameDarkAqua : NSAppearanceNameAqua];
    if (_focusMode) [self applyFocusMode];
    [_text setNeedsDisplay:YES];
    _suppress = wasSuppress;
}
- (NSAttributedString *)attributedRuns:(NSArray *)runs paragraph:(NSDictionary *)paragraph extra:(NSDictionary *)extra {
    NSMutableAttributedString *result = [NSMutableAttributedString new];
    for (NSDictionary *run in runs) {
        NSMutableDictionary *attrs = [@{CRParagraph:paragraph} mutableCopy];
        [attrs addEntriesFromDictionary:extra];
        if ([run[@"break"] boolValue]) {
            // U+2028 is a line separator: it wraps visually but keeps the whole
            // block in one -paragraphRangeForRange:, so block identity holds.
            [result appendAttributedString:[[NSAttributedString alloc] initWithString:@" " attributes:[self visual:attrs paragraph:paragraph]]];
            continue;
        }
        if ([run[@"math"] length]) {
            NSAttributedString *m=[self inlineMath:run paragraph:paragraph extra:extra];
            if (m) { [result appendAttributedString:m]; continue; }
        }
        if ([run[@"bold"] boolValue]) attrs[CRBold] = @YES;
        if ([run[@"italic"] boolValue]) attrs[CRItalic] = @YES;
        if ([run[@"code"] boolValue]) attrs[CRCode] = @YES;
        if ([run[@"strike"] boolValue]) attrs[NSStrikethroughStyleAttributeName] = @(NSUnderlineStyleSingle);
        if ([run[@"underline"] boolValue]) attrs[CRUnderline] = @YES;
        if ([run[@"highlight"] boolValue]) attrs[CRHighlight] = @YES;
        if ([run[@"sup"] boolValue]) attrs[CRSup] = @YES;
        if ([run[@"sub"] boolValue]) attrs[CRSub] = @YES;
        if ([run[@"link"] length]) attrs[NSLinkAttributeName] = run[@"link"];
        [result appendAttributedString:[[NSAttributedString alloc] initWithString:run[@"text"] ?: @"" attributes:[self visual:attrs paragraph:paragraph]]];
    }
    return result;
}
- (NSAttributedString *)attributedDocument:(NSDictionary *)document {
    NSMutableAttributedString *result = [NSMutableAttributedString new];
    NSArray *paragraphs = document[@"paragraphs"];
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
        if (CRKind(p,@"math") || CRKind(p,@"image") || CRKind(p,@"diagram")) {
            [result appendAttributedString:[self figure:p]];
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
    return result;
}
- (void)normalize {
    // NSTextView inherits attributes when a paragraph splits. Keep source
    // metadata once, then give newly created paragraphs fresh semantic identity.
    NSMutableSet *seen = [NSMutableSet set];
    for (NSValue *v in CRRanges(_text.string)) {
        NSRange range = v.rangeValue;
        NSDictionary *p = CRParagraphAt(self,range.location);
        if (CRKind(p,@"table")) continue;
        NSString *identifier = p[@"id"];
        if (identifier && [seen containsObject:identifier]) p = CRFresh(p);
        if (identifier) [seen addObject:identifier];
        if (range.length) [_text.textStorage addAttribute:CRParagraph value:p range:range];
        else { NSMutableDictionary *typing = [_text.typingAttributes mutableCopy]; typing[CRParagraph]=p; _text.typingAttributes=typing; _emptyParagraph=p; }
    }
    [self restyle];
}
- (NSDictionary *)document { return [self documentForRange:NSMakeRange(NSNotFound,0)]; }
- (NSDictionary *)documentForRange:(NSRange)selection {
    NSMutableArray *paragraphs = [NSMutableArray array];
    NSTextTable *currentTable = nil;
    NSMutableDictionary *tableParagraph = nil;
    for (NSValue *v in CRRanges(_text.string)) {
        NSRange range = CRContent(_text.string,v.rangeValue);
        if (range.length == 0 && range.location == _text.string.length && range.location > 0 && CRCellAt(self,range.location-1)) continue;
        NSTextTableBlock *cell = CRCellAt(self,range.location);
        if (selection.location != NSNotFound) { range=NSIntersectionRange(range,selection); if (!range.length) continue; }
        NSMutableDictionary *p = [CRParagraphAt(self,range.location) mutableCopy];
        BOOL figure=CRKind(p,@"math") || CRKind(p,@"image") || CRKind(p,@"diagram");
        if (figure && ![[_text.string substringWithRange:range] isEqual:@"\uFFFC"]) {
            // The user typed into the figure: it is source text now.
            p[@"kind"]=@"source";
            figure=NO;
        }
        if (!CRKind(p,@"opaque") && !figure) {
            NSMutableArray *runs = [NSMutableArray array];
            [_text.textStorage enumerateAttributesInRange:range options:0 usingBlock:^(NSDictionary *attrs, NSRange r, BOOL *stop) {
                if (attrs[CRMathText]) {
                    [runs addObject:@{@"text":attrs[CRMathText],@"math":attrs[CRMathExpr]}];
                    return;
                }
                NSString *chunk=[[self.text.string substringWithRange:r] stringByReplacingOccurrencesOfString:@"\uFFFC" withString:@""];
                if (!chunk.length) return;
                NSMutableDictionary *base = [NSMutableDictionary dictionary];
                if ([attrs[CRBold] boolValue]) base[@"bold"]=@YES;
                if ([attrs[CRItalic] boolValue]) base[@"italic"]=@YES;
                if ([attrs[CRCode] boolValue]) base[@"code"]=@YES;
                if ([attrs[NSStrikethroughStyleAttributeName] integerValue]) base[@"strike"]=@YES;
                if ([attrs[CRUnderline] boolValue]) base[@"underline"]=@YES;
                if ([attrs[CRHighlight] boolValue]) base[@"highlight"]=@YES;
                if ([attrs[CRSup] boolValue]) base[@"sup"]=@YES;
                if ([attrs[CRSub] boolValue]) base[@"sub"]=@YES;
                if (attrs[NSLinkAttributeName]) base[@"link"]=[attrs[NSLinkAttributeName] description];
                // U+2028 line separators become break runs so a soft visual
                // line survives the round-trip as a Markdown hard break.
                NSArray *pieces=[chunk componentsSeparatedByString:@"\u2028"];
                for (NSUInteger pi=0; pi<pieces.count; pi++) {
                    if (pi) [runs addObject:@{@"break":@YES}];
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
            NSRange tableRange = CRTableRange(self,cell.table);
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
            tableParagraph[@"rows"][cell.startingRow][cell.startingColumn] = @{@"runs":p[@"runs"]};
            continue;
        }
        currentTable = nil;
        [paragraphs addObject:p];
    }
    return @{@"paragraphs":paragraphs};
}
- (BOOL)allowRange:(NSRange)range {
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
- (BOOL)textView:(NSTextView *)textView shouldChangeTextInRange:(NSRange)range replacementString:(NSString *)text {
    return _suppress || [self allowRange:range];
}
- (void)textDidChange:(NSNotification *)notification {
    if (_suppress) return;
    if (!_text.hasMarkedText) [self normalize];
    if (_typewriter) [self centerCaretLine];
    [self notify];
}
- (void)textViewDidChangeSelection:(NSNotification *)notification {
    [self updateTypingParagraph];
    [self finishDraftGroup];
    if (_suppress || _text.hasMarkedText) return;
    if (_focusMode) [self restyle]; // the dimmed paragraphs follow the caret
    if (_typewriter) [self centerCaretLine];
}
- (void)textDidEndEditing:(NSNotification *)notification {
    // Leaving the editor finishes any open draft group: the block the caret was
    // editing re-renders to its figure form (a formula/image/code), regardless
    // of where the caret sat inside it.
    if (_suppress || _text.hasMarkedText) return;
    NSRange group=_draftGroup;
    _draftGroup=NSMakeRange(NSNotFound,0);
    if (group.location!=NSNotFound) [self renderDraftGroup:group];
}
- (void)updateTypingParagraph {
    if (_suppress || _text.hasMarkedText || _text.selectedRange.length) return;
    NSUInteger location = _text.selectedRange.location;
    NSDictionary *p = CRParagraphAt(self,location);
    NSMutableDictionary *typing = [_text.typingAttributes mutableCopy];
    typing[CRParagraph] = p;
    NSTextTableBlock *cell = CRCellAt(self,location);
    if (cell) typing[CRTableCell] = cell; else [typing removeObjectForKey:CRTableCell];
    _text.typingAttributes = [self visual:typing paragraph:p];
}
- (void)restore:(NSAttributedString *)value selection:(NSRange)selection typing:(NSDictionary *)typing ending:(NSDictionary *)ending {
    NSAttributedString *before = [_text.textStorage copy]; NSRange oldSelection = _text.selectedRange; NSDictionary *oldTyping = _text.typingAttributes; NSDictionary *oldEnding=_emptyParagraph;
    [_undo registerUndoWithTarget:self handler:^(CREditor *target) { [target restore:before selection:oldSelection typing:oldTyping ending:oldEnding]; }];
    _suppress=YES;
    [_text.textStorage setAttributedString:value]; _text.selectedRange=selection; _text.typingAttributes=typing; _emptyParagraph=ending;
    _suppress=NO;
    [self restyle]; [self notify];
}
- (void)replace:(NSRange)range with:(NSAttributedString *)value selection:(NSRange)selection {
    if (!_text.editable || _text.hasMarkedText || ![self allowRange:range]) return;
    [_text breakUndoCoalescing];
    NSAttributedString *before=[_text.textStorage copy]; NSRange oldSelection=_text.selectedRange; NSDictionary *oldTyping=_text.typingAttributes; NSDictionary *oldEnding=_emptyParagraph;
    [_undo beginUndoGrouping];
    [_undo registerUndoWithTarget:self handler:^(CREditor *target) { [target restore:before selection:oldSelection typing:oldTyping ending:oldEnding]; }];
    BOOL wasSuppress=_suppress; _suppress=YES;
    [_text.textStorage replaceCharactersInRange:range withAttributedString:value]; _text.selectedRange=selection;
    _suppress=wasSuppress;
    [self normalize];
    [_undo endUndoGrouping];
    [self notify];
}
- (void)format:(NSString *)command {
    if (!_text.editable || _text.hasMarkedText) return;
    NSRange selection = _text.selectedRange;
    for (NSValue *v in CRSelectedParagraphs(self)) if (CRKind(CRParagraphAt(self,v.rangeValue.location),@"opaque")) { [self source]; return; }
    if ([command hasPrefix:@"link:"]) {
        // link:URL wraps the selection in a link run; empty selection no-ops.
        NSString *url=[command substringFromIndex:5];
        if (!selection.length || !url.length) return;
        NSMutableAttributedString *after=[_text.textStorage mutableCopy];
        [after addAttribute:NSLinkAttributeName value:url range:selection];
        [self replace:NSMakeRange(0,_text.string.length) with:after selection:selection];
        return;
    }
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
        for (NSValue *v in CRSelectedParagraphs(self)) {
            NSRange r=v.rangeValue;
            NSMutableDictionary *p=[CRParagraphAt(self,r.location) mutableCopy];
            p[@"kind"]=kind;
            [p removeObjectForKey:@"level"]; [p removeObjectForKey:@"checked"]; [p removeObjectForKey:@"ordinal"];
            if ([kind isEqual:@"heading"]) p[@"level"]=@(level);
            if ([kind isEqual:@"ordered"]) p[@"ordinal"]=@1;
            if (r.length) [after addAttribute:CRParagraph value:p range:r];
            else { NSMutableDictionary *typing=[_text.typingAttributes mutableCopy]; typing[CRParagraph]=p; newTyping=[self visual:typing paragraph:p]; }
        }
        if (clear) {
            // Clear strips the inline traits as well as the paragraph kind.
            NSArray *traits=@[CRBold,CRItalic,CRCode,CRUnderline,CRHighlight,CRSup,CRSub,NSStrikethroughStyleAttributeName,NSLinkAttributeName];
            if (selection.length) for (NSString *trait in traits) [after removeAttribute:trait range:selection];
            else {
                NSMutableDictionary *p=[CRParagraphAt(self,selection.location) mutableCopy];
                p[@"kind"]=@"body"; [p removeObjectForKey:@"level"]; [p removeObjectForKey:@"checked"]; [p removeObjectForKey:@"ordinal"];
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
}
- (void)newline {
    if (!_text.editable || _text.hasMarkedText) { [_text insertText:@"\n" replacementRange:_text.selectedRange]; return; }
    [self emojiRule:NO]; // Enter closes a trailing :emoji: before the split
    NSRange selection=_text.selectedRange;
    if (CRCellAt(self,selection.location)) { [self moveTableCell:NO]; return; }
    NSRange paragraph=[_text.string paragraphRangeForRange:NSMakeRange(selection.location,0)];
    NSDictionary *p=CRParagraphAt(self,paragraph.location);
    BOOL list=CRKind(p,@"checklist") || CRKind(p,@"bullet") || CRKind(p,@"ordered");
    // Enter on an empty list item or quote line leaves the block (Typora).
    if ((list || CRKind(p,@"quote")) && CRContent(_text.string,paragraph).length==0) { [self format:@"body"]; return; }
    if (CRKind(p,@"opaque")) { [self source]; return; }
    NSRange content=CRContent(_text.string,paragraph);
    if (CRKind(p,@"code")) {
        // Enter on the empty last line leaves the block, like Typora.
        NSUInteger nextParagraph=NSMaxRange(paragraph);
        BOOL lastLine=nextParagraph>=_text.string.length || !CRKind(CRParagraphAt(self,nextParagraph),@"code");
        if (lastLine && content.length==0) { [self format:@"body"]; return; }
    }
    if (CRKind(p,@"body")) {
        // ``` followed by Enter opens a code block, the way Typora does.
        NSString *text=[_text.string substringWithRange:content];
        NSTextCheckingResult *fence=[[NSRegularExpression regularExpressionWithPattern:@"^(```|~~~)([A-Za-z0-9_+.\\-]*)$" options:0 error:NULL] firstMatchInString:text options:0 range:NSMakeRange(0,text.length)];
        if (fence) {
            NSString *mark=[text substringWithRange:[fence rangeAtIndex:1]];
            NSMutableDictionary *cp=[@{@"kind":@"code",@"open":[mark stringByAppendingString:[text substringWithRange:[fence rangeAtIndex:2]]],@"close":mark,@"runs":@[]} mutableCopy];
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
        NSDictionary *body=CRBody();
        NSAttributedString *line=[[NSAttributedString alloc] initWithString:@"\n" attributes:[self visual:@{CRParagraph:body} paragraph:body]];
        _suppress=YES;
        [self replace:selection with:line selection:NSMakeRange(selection.location+1,0)];
        _text.typingAttributes=[self visual:@{CRParagraph:p} paragraph:p];
        _suppress=NO; [self restyle]; [self renumber]; [self notify];
        return;
    }
    NSDictionary *next=(list || CRKind(p,@"quote") || CRKind(p,@"source") || CRKind(p,@"code")) ? CRFresh(p) : CRBody();
    NSDictionary *attrs=[self visual:@{CRParagraph:p} paragraph:p];
    NSAttributedString *newline=[[NSAttributedString alloc] initWithString:@"\n" attributes:attrs];
    _suppress=YES;
    [self replace:selection with:newline selection:NSMakeRange(selection.location+1,0)];
    NSRange nextRange=[_text.string paragraphRangeForRange:NSMakeRange(selection.location+1,0)];
    if (nextRange.length) [_text.textStorage addAttribute:CRParagraph value:next range:nextRange];
    _text.typingAttributes=[self visual:@{CRParagraph:next} paragraph:next]; _emptyParagraph=next;
    _suppress=NO; [self renumber]; [self restyle]; [self notify];
}
// renumber keeps each run of ordered items counting up from its first
// ordinal, so inserting or removing an item never leaves a duplicate number.
- (void)renumber {
    NSInteger expected=0;
    BOOL wasSuppress=_suppress; _suppress=YES;
    for (NSValue *v in CRRanges(_text.string)) {
        NSRange r=v.rangeValue;
        NSDictionary *p=CRParagraphAt(self,r.location);
        if (!CRKind(p,@"ordered")) { expected=0; continue; }
        NSInteger ordinal=[p[@"ordinal"] integerValue];
        if (!expected) { expected=MAX(ordinal,1)+1; continue; }
        if (ordinal!=expected) {
            NSMutableDictionary *updated=[p mutableCopy]; updated[@"ordinal"]=@(expected);
            if (r.length) [_text.textStorage addAttribute:CRParagraph value:updated range:r];
            else { NSMutableDictionary *typing=[_text.typingAttributes mutableCopy]; typing[CRParagraph]=updated; _text.typingAttributes=typing; _emptyParagraph=updated; }
        }
        expected++;
    }
    _suppress=wasSuppress;
    [_text setNeedsDisplay:YES];
}
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
                _text.selectedRange = NSMakeRange(next.location,0);
                [_text scrollRangeToVisible:_text.selectedRange];
                return YES;
            }
        }
        if (backward) { _text.selectedRange=NSMakeRange(range.location,0); return YES; }
        NSRange tableRange = CRTableRange(self,cell.table);
        NSMutableDictionary *paragraph = [[self documentForRange:tableRange][@"paragraphs"][0] mutableCopy];
        NSMutableArray *rows = [paragraph[@"rows"] mutableCopy];
        NSMutableArray *row = [NSMutableArray array];
        for (NSInteger column=0; column<cell.table.numberOfColumns; column++) [row addObject:@{@"runs":@[]}];
        [rows addObject:row]; paragraph[@"rows"] = rows;
        NSDictionary *ending = _emptyParagraph;
        NSAttributedString *value = [self attributedDocument:@{@"paragraphs":@[paragraph]}];
        _emptyParagraph = ending;
        [self replace:tableRange with:value selection:NSMakeRange(NSMaxRange(tableRange),0)];
        [self updateTypingParagraph];
        [_text scrollRangeToVisible:_text.selectedRange];
        return YES;
    }
    return YES;
}
- (void)applyInputRules {
    if (_text.hasMarkedText || !_text.editable || _text.selectedRange.length) return;
    NSUInteger caret = _text.selectedRange.location;
    NSRange paragraphRange = [_text.string paragraphRangeForRange:NSMakeRange(caret,0)];
    NSDictionary *paragraph = CRParagraphAt(self,paragraphRange.location);
    if (CRKind(paragraph,@"opaque") || CRKind(paragraph,@"source") || CRKind(paragraph,@"code")) return;
    NSRange beforeCaret = NSMakeRange(paragraphRange.location,caret-paragraphRange.location);
    NSString *prefix = [_text.string substringWithRange:beforeCaret];
    if (_text.typingAttributes[CRCode] || (caret && [[_text.textStorage attribute:CRCode atIndex:caret-1 effectiveRange:NULL] boolValue])) return;
    if ([self emojiRule:YES]) return;
    NSString *kind = nil;
    NSInteger level=0, ordinal=0;
    BOOL checked=NO;
    if (CRKind(paragraph,@"body")) {
        NSTextCheckingResult *heading = [[NSRegularExpression regularExpressionWithPattern:@"^(#{1,6}) $" options:0 error:NULL] firstMatchInString:prefix options:0 range:NSMakeRange(0,prefix.length)];
        NSTextCheckingResult *ordered = [[NSRegularExpression regularExpressionWithPattern:@"^([0-9]{1,9})[.)] $" options:0 error:NULL] firstMatchInString:prefix options:0 range:NSMakeRange(0,prefix.length)];
        if (heading) { kind=@"heading"; level=[heading rangeAtIndex:1].length; }
        else if (ordered) { kind=@"ordered"; ordinal=[[prefix substringWithRange:[ordered rangeAtIndex:1]] integerValue]; }
        else if ([@[@"- ",@"* ",@"+ "] containsObject:prefix]) kind=@"bullet";
        else if ([prefix isEqual:@"> "]) kind=@"quote";
    }
    if ((CRKind(paragraph,@"body") || CRKind(paragraph,@"bullet")) && ([@[@"[ ] ",@"[x] ",@"[X] "] containsObject:prefix])) {
        kind=@"checklist"; checked=![[prefix substringWithRange:NSMakeRange(1,1)] isEqual:@" "];
    }
    if (kind) {
        NSMutableDictionary *updated = [paragraph mutableCopy]; updated[@"kind"]=kind;
        [updated removeObjectForKey:@"level"]; [updated removeObjectForKey:@"ordinal"]; [updated removeObjectForKey:@"checked"];
        if (level) updated[@"level"]=@(level);
        if ([kind isEqual:@"ordered"]) updated[@"ordinal"]=@(ordinal);
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
    NSArray *patterns = @[@"\\$[^$\\n]+\\$$", @"\\*\\*[^*\\n]+\\*\\*$", @"__[^_\\n]+__$", @"~~[^~\\n]+~~$", @"`[^`\\n]+`$", @"\\*[^*\\n]+\\*$", @"_[^_\\n]+_$", @"\\[[^\\]\\n]+\\]\\([^\\s)]+\\)$"];
    for (NSString *pattern in patterns) {
        NSTextCheckingResult *match = [[NSRegularExpression regularExpressionWithPattern:pattern options:0 error:NULL] firstMatchInString:prefix options:0 range:NSMakeRange(0,prefix.length)];
        if (!match) continue;
        NSUInteger start=match.range.location, slashCount=0;
        while (start > slashCount && [prefix characterAtIndex:start-slashCount-1]=='\\') slashCount++;
        if (slashCount % 2 || (start && [prefix characterAtIndex:start-1]==[prefix characterAtIndex:start])) continue;
        if (start && [prefix characterAtIndex:start]=='_' && [NSCharacterSet.alphanumericCharacterSet characterIsMember:[prefix characterAtIndex:start-1]]) continue;
        NSString *source = [prefix substringWithRange:match.range];
        char *json = cr_parse_markdown((char *)source.UTF8String);
        if (!json) continue;
        NSDictionary *doc = [NSJSONSerialization JSONObjectWithData:[[NSString stringWithUTF8String:json] dataUsingEncoding:NSUTF8StringEncoding] options:0 error:NULL]; free(json);
        NSArray *paragraphs = doc[@"paragraphs"];
        if (paragraphs.count != 1 || !CRKind(paragraphs[0],@"body")) continue;
        NSArray *runs = paragraphs[0][@"runs"];
        BOOL styled=NO;
        for (NSDictionary *run in runs) styled |= [run[@"bold"] boolValue] || [run[@"italic"] boolValue] || [run[@"strike"] boolValue] || [run[@"code"] boolValue] || [run[@"link"] length] > 0 || [run[@"math"] length] > 0;
        if (!styled) continue;
        NSRange range=NSMakeRange(beforeCaret.location+match.range.location,match.range.length);
        NSTextTableBlock *cell=CRCellAt(self,range.location);
        NSDictionary *extra=cell ? @{CRTableCell:cell} : @{};
        NSAttributedString *value=[self attributedRuns:runs paragraph:paragraph extra:extra];
        NSDictionary *typing=_text.typingAttributes;
        [self replace:range with:value selection:NSMakeRange(range.location+value.length,0)];
        _text.typingAttributes=typing;
        return;
    }
}
- (NSRect)markerLineRect:(NSRange)range {
    [_text.layoutManager ensureLayoutForTextContainer:_text.textContainer];
    NSRange glyph=[_text.layoutManager glyphRangeForCharacterRange:CRContent(_text.string,range) actualCharacterRange:NULL];
    if (!glyph.length) return NSOffsetRect(_text.layoutManager.extraLineFragmentRect,_text.textContainerOrigin.x,_text.textContainerOrigin.y);
    // Tight glyph bounds of the first line: the fragment rect includes
    // lineSpacing below the glyphs (markers ride low) and the used rect
    // starts at the indented first glyph (markers shift right onto text).
    NSRange lineGlyphs;
    [_text.layoutManager lineFragmentRectForGlyphAtIndex:glyph.location effectiveRange:&lineGlyphs];
    NSRect bounds=[_text.layoutManager boundingRectForGlyphRange:lineGlyphs inTextContainer:_text.textContainer];
    return NSOffsetRect(bounds,_text.textContainerOrigin.x,_text.textContainerOrigin.y);
}
- (NSRect)markerRect:(NSRange)range {
    NSRect line=[self markerLineRect:range];
    return NSMakeRect(_text.textContainerOrigin.x+4*_zoom,NSMidY(line)-9.5*_zoom,19*_zoom,19*_zoom);
}
- (void)toggleCheckbox:(NSUInteger)location {
    if (!_text.editable || _text.hasMarkedText) return;
    NSRange range=[_text.string paragraphRangeForRange:NSMakeRange(location,0)];
    NSMutableDictionary *p=[CRParagraphAt(self,location) mutableCopy];
    if (!CRKind(p,@"checklist")) return;
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
        NSDictionary *lp=@{@"kind":@"source",@"runs":@[@{@"text":text}],@"raw":text,@"separator":sep};
        [value appendAttributedString:[[NSAttributedString alloc] initWithString:text attributes:[self visual:@{CRParagraph:lp} paragraph:lp]]];
        if (i+1<lines.count) [value appendAttributedString:[[NSAttributedString alloc] initWithString:@"\n" attributes:[self visual:@{CRParagraph:lp} paragraph:lp]]];
    }
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

- (NSRect)imageBounds:(NSImage *)img {
    CGFloat w=img.size.width,h=img.size.height;
    if (w<=0 || h<=0) return NSMakeRect(0,0,1,1);
    CGFloat maxW=MAX(60,_text.textContainer.size.width-8*_zoom);
    if (w>maxW) { h=h*maxW/w; w=maxW; }
    return NSMakeRect(0,0,w,h);
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
    BOOL (^drafty)(NSInteger)=^BOOL(NSInteger i){
        NSDictionary *p=CRParagraphAt(self,[ranges[i] rangeValue].location);
        return CRKind(p,@"source") || CRKind(p,@"body");
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
    char *json=cr_parse_markdown((char *)text.UTF8String);
    if (!json) return NO;
    NSMutableDictionary *doc=[[NSJSONSerialization JSONObjectWithData:[[NSString stringWithUTF8String:json] dataUsingEncoding:NSUTF8StringEncoding] options:0 error:NULL] mutableCopy];
    free(json);
    NSArray *ps=doc[@"paragraphs"];
    if (!ps.count) return NO;
    BOOL renderable=YES;
    NSMutableArray *paragraphs=[NSMutableArray array];
    for (NSDictionary *p in ps) {
        if (!CRKind(p,@"math") && !CRKind(p,@"image") && !CRKind(p,@"code") && !CRKind(p,@"diagram")) renderable=NO;
        // Fresh parse IDs collide with the document's; identity stays valid
        // through baseline, not id.
        NSMutableDictionary *clean=[p mutableCopy];
        [clean removeObjectForKey:@"id"]; [clean removeObjectForKey:@"nextId"];
        [paragraphs addObject:clean];
    }
    if (!renderable) return NO;
    doc[@"paragraphs"]=paragraphs;
    NSDictionary *ending=_emptyParagraph;
    NSAttributedString *value=[self attributedDocument:doc];
    _emptyParagraph=ending;
    // A single figure that falls back to source text changes nothing.
    if (ps.count==1 && ![value attribute:NSAttachmentAttributeName atIndex:0 effectiveRange:NULL]) return NO;
    NSRange content=CRContent(_text.string,range);
    [self replace:content with:value selection:NSMakeRange(NSMaxRange(content),0)];
    return YES;
}

// renderDraftGroup re-renders the group the caret just left, if it parses as
// renderable blocks.
- (void)renderDraftGroup:(NSRange)group {
    [self renderSource:[_text.string substringWithRange:CRContent(_text.string,group)] inRange:group];
}

// finishDraftGroup re-renders the block the caret just left, if any.
- (void)finishDraftGroup {
    if (_suppress || _text.hasMarkedText) return;
    NSRange group=_draftGroup;
    _draftGroup=NSMakeRange(NSNotFound,0);
    NSUInteger caret=_text.selectedRange.location;
    if (group.location!=NSNotFound && (caret<group.location || caret>NSMaxRange(group))) [self renderDraftGroup:group];
    _draftGroup=[self draftGroupAt:_text.selectedRange.location];
}

// the text baseline; nil falls back to the literal source text.
- (NSAttributedString *)inlineMath:(NSDictionary *)run paragraph:(NSDictionary *)paragraph extra:(NSDictionary *)extra {
    NSString *expr=run[@"math"];
    NSString *key=[NSString stringWithFormat:@"i|%d|%f|%@",_dark,_zoom,expr];
    id cached=_mathCache[key];
    NSImage *img=nil;
    if ([cached isKindOfClass:NSImage.class]) img=cached;
    else if (cached) return nil;
    else {
        char *b64=cr_render_math((char *)[expr UTF8String],_dark?1:0,_zoom,1);
        NSData *data=b64 ? [[NSData alloc] initWithBase64EncodedString:[NSString stringWithUTF8String:b64] options:0] : nil;
        if (b64) free(b64);
        img=data.length ? [[NSImage alloc] initWithData:data] : nil;
        if (img) { img.size=NSMakeSize(img.size.width/2,img.size.height/2); _mathCache[key]=img; }
        else { _mathCache[key]=[NSNull null]; return nil; }
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
    [s addAttributes:[self visual:attrs paragraph:paragraph] range:NSMakeRange(0,s.length)];
    // visual() rebuilds the attribute dictionary; keep the identity keys.
    [s addAttributes:@{CRMathText:run[@"text"],CRMathExpr:expr} range:NSMakeRange(0,s.length)];
    return s;
}

// figure renders a math or image paragraph as an attachment; anything that
// cannot render becomes its own editable source text instead.
- (NSAttributedString *)figure:(NSDictionary *)p {
    NSString *raw=[p[@"raw"] isKindOfClass:NSString.class] ? p[@"raw"] : @"";
    NSImage *img=nil;
    BOOL failed=NO;
    NSTextAttachment *pending=nil;
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
        NSString *key=[NSString stringWithFormat:@"%d|%f|%@",_dark,_zoom,raw];
        id cached=_mathCache[key];
        if ([cached isKindOfClass:NSImage.class]) img=cached;
        else if (cached) failed=YES;
        else {
            char *b64=cr_render_math((char *)raw.UTF8String,_dark?1:0,_zoom,0);
            NSData *data=b64 ? [[NSData alloc] initWithBase64EncodedString:[NSString stringWithUTF8String:b64] options:0] : nil;
            if (b64) free(b64);
            img=data.length ? [[NSImage alloc] initWithData:data] : nil;
            if (img) { img.size=NSMakeSize(img.size.width/2,img.size.height/2); _mathCache[key]=img; } // RenderMath draws at 2x
            else { _mathCache[key]=[NSNull null]; failed=YES; }
        }
    } else {
        NSTextCheckingResult *m=[[NSRegularExpression regularExpressionWithPattern:@"!\\[[^\\]]*\\]\\(([^\\s)]+)" options:0 error:NULL] firstMatchInString:raw options:0 range:NSMakeRange(0,raw.length)];
        NSString *url=m ? [[raw substringWithRange:[m rangeAtIndex:1]] stringByRemovingPercentEncoding] : nil;
        if (!url.length) failed=YES;
        else {
            id cached=_imageCache[url];
            if ([cached isKindOfClass:NSImage.class]) img=cached;
            else if (cached) failed=YES;
            else if ([url hasPrefix:@"http://"] || [url hasPrefix:@"https://"]) {
                img=[self placeholderImage];
                if (!_imagePending[url]) {
                    _imagePending[url]=@YES;
                    [[[NSURLSession sharedSession] dataTaskWithURL:[NSURL URLWithString:url] completionHandler:^(NSData *data,NSURLResponse *response,NSError *error) {
                        NSImage *loaded=data.length ? [[NSImage alloc] initWithData:data] : nil;
                        dispatch_async(dispatch_get_main_queue(),^{
                            self.imageCache[url]=loaded ?: (id)[NSNull null];
                            [self.imagePending removeObjectForKey:url];
                            for (NSTextAttachment *att in [self.imageAttachments[url] objectEnumerator]) if (loaded) { att.image=loaded; att.bounds=[self imageBounds:loaded]; }
                            [self.text.layoutManager invalidateLayoutForCharacterRange:NSMakeRange(0,self.text.textStorage.length) actualCharacterRange:NULL];
                            [self.text setNeedsDisplay:YES];
                        });
                    }] resume];
                }
                NSTextAttachment *att=[NSTextAttachment new];
                att.image=img; att.bounds=[self imageBounds:img];
                NSMutableArray *list=_imageAttachments[url];
                if (!list) { list=[NSMutableArray new]; _imageAttachments[url]=list; }
                [list addObject:att];
                pending=att;
            } else {
                NSString *path=url;
                if (![path hasPrefix:@"/"] && _baseDir.length) path=[_baseDir stringByAppendingPathComponent:path];
                img=[[NSImage alloc] initWithContentsOfFile:path];
                if (img) _imageCache[url]=img;
                else { _imageCache[url]=[NSNull null]; failed=YES; }
            }
        }
    }
    if (failed) return [self sourceText:p];
    NSTextAttachment *att=pending;
    if (!att) {
        att=[NSTextAttachment new];
        att.image=img;
        att.bounds=[self imageBounds:img];
    }
    if (!att.image) return [self sourceText:p];
    NSMutableAttributedString *s=[[NSAttributedString attributedStringWithAttachment:att] mutableCopy];
    NSMutableDictionary *attrs=[[self visual:@{CRParagraph:p} paragraph:p] mutableCopy];
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
    if (effective.location==NSNotFound) effective = self.markedRange;
    if (effective.location==NSNotFound) effective = self.selectedRange;
    if (![self.owner allowRange:effective]) return;
    BOOL opaque=NO;
    for (NSValue *v in CRRanges(self.string)) if (NSIntersectionRange(v.rangeValue,effective).length && CRKind(CRParagraphAt(self.owner,v.rangeValue.location),@"opaque")) opaque=YES;
    NSTextTableBlock *targetCell=CRCellAt(self.owner,effective.location);
    if (targetCell && effective.length) {
        NSRange tableRange=CRTableRange(self.owner,targetCell.table);
        if (effective.location <= tableRange.location && NSMaxRange(effective) >= NSMaxRange(tableRange)) opaque=YES;
    }
    if (opaque) {
        if (![self.owner allowRange:effective]) return;
        NSString *string=[value isKindOfClass:NSAttributedString.class] ? [value string] : value;
        NSDictionary *attrs=[self.owner visual:@{CRParagraph:CRBody()} paragraph:CRBody()];
        self.owner.suppress=YES;
        [self.owner replace:effective with:[[NSAttributedString alloc] initWithString:string attributes:attrs] selection:NSMakeRange(effective.location+string.length,0)];
        self.typingAttributes=attrs; self.owner.emptyParagraph=CRBody(); self.owner.suppress=NO; [self.owner notify];
        return;
    }
    if ([self.owner autoPair:value range:effective]) return;
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
        [self.owner normalize];
        NSString *string=[value isKindOfClass:NSAttributedString.class] ? [value string] : value;
        if (string.length && [string rangeOfCharacterFromSet:NSCharacterSet.newlineCharacterSet].location==NSNotFound) [self.owner applyInputRules];
    }
    self.owner.suppress=wasSuppress;
    if (!wasSuppress) [self.owner notify];
}
- (void)insertNewline:(id)sender { [self.owner newline]; }
- (void)insertLineBreak:(id)sender {
    // Shift+Return authors a Markdown hard break: a soft visual line (U+2028)
    // inside the current block, not a new paragraph. Tables keep their own
    // row/newline semantics, so fall back there.
    if (CRCellAt(self.owner,self.selectedRange.location)) { [self.owner newline]; return; }
    NSDictionary *p=CRParagraphAt(self.owner,self.selectedRange.location);
    if (CRKind(p,@"code") || CRKind(p,@"source") || CRKind(p,@"opaque") || CRKind(p,@"math") || CRKind(p,@"image")) { [self.owner newline]; return; }
    unichar ls=0x2028;
    [self insertText:[NSString stringWithCharacters:&ls length:1] replacementRange:self.selectedRange];
}
- (void)insertTab:(id)sender {
    if ([self.owner moveTableCell:NO]) return;
    // Nested lists are not editable yet; a literal tab would only turn the
    // item into "&#9;" text, so Tab stays inert inside list items.
    NSDictionary *p=CRParagraphAt(self.owner,self.selectedRange.location);
    if (CRKind(p,@"bullet") || CRKind(p,@"ordered") || CRKind(p,@"checklist")) return;
    [super insertTab:sender];
}
- (void)insertBacktab:(id)sender {
    if ([self.owner moveTableCell:YES]) return;
    NSDictionary *p=CRParagraphAt(self.owner,self.selectedRange.location);
    if (CRKind(p,@"bullet") || CRKind(p,@"ordered") || CRKind(p,@"checklist")) { [self.owner format:@"body"]; return; }
    [super insertBacktab:sender];
}
- (void)deleteBackward:(id)sender {
    // Between an empty auto-pair, one Backspace removes both characters.
    CREditor *e=self.owner;
    NSRange selection=self.selectedRange;
    if (!e.suppress && self.editable && !self.hasMarkedText && selection.length==0) {
        // Backspace at the start of a heading, list item or quote turns it
        // back into body text first, the way Typora does.
        NSRange paragraph=[self.string paragraphRangeForRange:NSMakeRange(selection.location,0)];
        NSDictionary *p=CRParagraphAt(e,paragraph.location);
        BOOL block=CRKind(p,@"heading") || CRKind(p,@"bullet") || CRKind(p,@"ordered") || CRKind(p,@"checklist") || CRKind(p,@"quote");
        if (block && selection.location==paragraph.location && !CRCellAt(e,selection.location)) { [e format:@"body"]; return; }
    }
    if (!e.suppress && self.editable && !self.hasMarkedText && selection.length==0 && selection.location>0 && selection.location<self.string.length) {
        unichar open=[self.string characterAtIndex:selection.location-1], close=[self.string characterAtIndex:selection.location];
        BOOL pair=(open=='('&&close==')') || (open=='['&&close==']') || (open=='{'&&close=='}') || (open=='"'&&close=='"') || (open=='\''&&close=='\'');
        if (pair) {
            NSDictionary *p=CRParagraphAt(e,selection.location);
            BOOL plain=!(CRKind(p,@"code") || CRKind(p,@"source") || CRKind(p,@"math") || CRKind(p,@"opaque")) && !CRCellAt(e,selection.location);
            NSDictionary *before=[self.textStorage attributesAtIndex:selection.location-1 effectiveRange:NULL];
            if (plain && !before[CRCode] && !before[CRMathText]) {
                [e replace:NSMakeRange(selection.location-1,2) with:[[NSAttributedString alloc] initWithString:@"" attributes:self.typingAttributes] selection:NSMakeRange(selection.location-1,0)];
                return;
            }
        }
    }
    [super deleteBackward:sender];
}
- (void)setMarkedText:(id)value selectedRange:(NSRange)selection replacementRange:(NSRange)replacement { [super setMarkedText:value selectedRange:selection replacementRange:replacement]; [self.owner notify]; }
- (void)unmarkText { [super unmarkText]; if (!self.owner.suppress) { [self.owner normalize]; [self.owner notify]; } }
- (void)drawRect:(NSRect)dirty {
    CREditor *e=self.owner;
    // drawsBackground is off: paint the canvas, then one rounded slab per
    // protected block, so the chips sit under the text and stay separated by
    // the real paragraph gap.
    [self.backgroundColor setFill];
    NSRectFill(dirty);
    __block NSRect group=NSZeroRect;
    void (^flush)(void)=^{
        if (NSIsEmptyRect(group)) return;
        // Set the fill here: drawing a language tag leaves its gray behind.
        [(e.dark ? [NSColor colorWithWhite:0.23 alpha:1] : [NSColor colorWithWhite:0.94 alpha:1]) setFill];
        NSRect chip=NSMakeRect(self.textContainerOrigin.x,group.origin.y-7*e.zoom,self.textContainer.size.width,group.size.height+14*e.zoom);
        if (NSIntersectsRect(chip,dirty)) [[NSBezierPath bezierPathWithRoundedRect:chip xRadius:6*e.zoom yRadius:6*e.zoom] fill];
        group=NSZeroRect;
    };
    __block NSString *groupLang=nil;
    void (^flushWithLabel)(void)=^{
        // Chip first, then the language tag on top of it.
        NSRect chip=NSIsEmptyRect(group) ? NSZeroRect : NSMakeRect(self.textContainerOrigin.x,group.origin.y-7*e.zoom,self.textContainer.size.width,group.size.height+14*e.zoom);
        flush();
        if (groupLang.length) {
            NSDictionary *attrs=@{NSFontAttributeName:[NSFont systemFontOfSize:10.5*e.zoom],NSForegroundColorAttributeName:e.dark ? [NSColor colorWithWhite:0.55 alpha:1] : [NSColor colorWithWhite:0.6 alpha:1]};
            NSSize size=[groupLang sizeWithAttributes:attrs];
            [groupLang drawAtPoint:NSMakePoint(NSMaxX(chip)-size.width-10*e.zoom,chip.origin.y+5*e.zoom) withAttributes:attrs];
        }
        groupLang=nil;
    };
    for (NSValue *v in CRRanges(self.string)) {
        NSDictionary *p=CRParagraphAt(e,v.rangeValue.location);
        // A blank or rich paragraph closes the block's chip.
        if (!CRKind(p,@"source") && !CRKind(p,@"code") && !CRKind(p,@"opaque")) { flushWithLabel(); continue; }
        [self.layoutManager ensureLayoutForTextContainer:self.textContainer];
        NSRange content=CRContent(self.string,v.rangeValue);
        if (!content.length && NSMaxRange(v.rangeValue)>content.location) content=NSMakeRange(content.location,1); // empty source line: the newline's own line
        NSRange glyph=[self.layoutManager glyphRangeForCharacterRange:content actualCharacterRange:NULL];
        if (!glyph.length) continue;
        NSRect r=[self.layoutManager boundingRectForGlyphRange:glyph inTextContainer:self.textContainer];
        r=NSOffsetRect(r,self.textContainerOrigin.x,self.textContainerOrigin.y);
        // A small vertical gap means the same block; more starts a new chip.
        if (!NSIsEmptyRect(group) && r.origin.y-NSMaxY(group) < 10*e.zoom) group=NSUnionRect(group,r);
        else {
            flushWithLabel();
            group=r;
            if (CRKind(p,@"code")) {
                // The language tag comes from the hidden opening fence.
                NSString *open=[p[@"open"] isKindOfClass:NSString.class] ? p[@"open"] : @"";
                groupLang=[open stringByTrimmingCharactersInSet:[NSCharacterSet characterSetWithCharactersInString:@"`~ 	"]];
            }
        }
        // A paragraph ending in a blank line closes its block.
        if (CRBlockEnding(p)) flushWithLabel();
    }
    flushWithLabel();
    [super drawRect:dirty];
    for (NSValue *v in CRRanges(self.string)) {
        NSDictionary *p=CRParagraphAt(e,v.rangeValue.location);
        NSRect rect=[e markerRect:v.rangeValue];
        if (!NSIntersectsRect(rect,dirty)) continue;
        // Markers match the body size and sit on the line's vertical center,
        // so bullets and ordinals read as part of the text, not as footnotes.
        NSRect lineRect=[e markerLineRect:v.rangeValue];
        NSColor *markerColor=e.dark ? [NSColor colorWithWhite:0.70 alpha:1] : [NSColor colorWithWhite:0.35 alpha:1];
        if (CRKind(p,@"checklist")) {
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
        else if (CRKind(p,@"quote")) { [(e.dark ? [NSColor colorWithWhite:0.40 alpha:1] : [NSColor colorWithWhite:0.80 alpha:1]) setFill]; [[NSBezierPath bezierPathWithRoundedRect:NSMakeRect(rect.origin.x+7,rect.origin.y,3*e.zoom,rect.size.height) xRadius:1.5*e.zoom yRadius:1.5*e.zoom] fill]; }
    }
}
- (void)mouseDown:(NSEvent *)event {
    if (self.window.firstResponder!=self) self.owner.previousResponder=self.window.firstResponder;
    NSPoint point=[self convertPoint:event.locationInWindow fromView:nil];
    for (NSValue *v in CRRanges(self.string)) {
        NSDictionary *p=CRParagraphAt(self.owner,v.rangeValue.location);
        if (CRKind(p,@"checklist") && NSPointInRect(point,[self.owner markerRect:v.rangeValue])) { [self.owner toggleCheckbox:v.rangeValue.location]; return; }
    }
    NSUInteger index=[self characterIndexForInsertionAtPoint:point];
    if (index < self.string.length && (CRKind(CRParagraphAt(self.owner,index),@"math") || CRKind(CRParagraphAt(self.owner,index),@"diagram"))) { [self.owner revealSourceAt:index]; return; }
    if (index < self.string.length && CRKind(CRParagraphAt(self.owner,index),@"opaque")) { [self.owner source]; return; }
    [super mouseDown:event];
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
    BOOL opaque=NO; for (NSDictionary *p in doc[@"paragraphs"]) opaque |= CRKind(p,@"opaque");
    if (opaque) {
        const char *json=[[NSString alloc] initWithData:data encoding:NSUTF8StringEncoding].UTF8String;
        char *source=cr_markdown_json((char *)json);
        if (source) { [NSPasteboard.generalPasteboard setString:[NSString stringWithUTF8String:source] forType:NSPasteboardTypeString]; free(source); }
    }
}
- (void)cut:(id)sender { if (![self.owner allowRange:self.selectedRange]) return; [self copy:sender]; [self insertText:@"" replacementRange:self.selectedRange]; }
- (void)paste:(id)sender {
    NSPasteboard *pb=NSPasteboard.generalPasteboard;
    NSTextTableBlock *cell=CRCellAt(self.owner,self.selectedRange.location);
    if (cell) {
        NSString *plain=[pb stringForType:NSPasteboardTypeString];
        if (plain) {
            plain=[[plain componentsSeparatedByCharactersInSet:NSCharacterSet.newlineCharacterSet] componentsJoinedByString:@" "];
            NSDictionary *paragraph=CRParagraphAt(self.owner,self.selectedRange.location);
            NSDictionary *attrs=[self.owner visual:@{CRParagraph:paragraph,CRTableCell:cell} paragraph:paragraph];
            [self.owner replace:self.selectedRange with:[[NSAttributedString alloc] initWithString:plain attributes:attrs] selection:NSMakeRange(self.selectedRange.location+plain.length,0)];
        }
        return;
    }
    NSData *own=[pb dataForType:CROwnPaste];
    if (own) {
        NSDictionary *document=[NSJSONSerialization JSONObjectWithData:own options:0 error:NULL];
        NSMutableArray *paragraphs=[NSMutableArray array];
        for (NSDictionary *original in document[@"paragraphs"]) { NSMutableDictionary *p=[original mutableCopy]; [p removeObjectForKey:@"id"]; [p removeObjectForKey:@"nextId"]; [paragraphs addObject:p]; }
        BOOL opaque=NO; for (NSDictionary *p in paragraphs) opaque |= CRKind(p,@"opaque") || CRKind(p,@"table") || CRKind(p,@"source");
        if (opaque && self.string.length) {
            NSData *data=[NSJSONSerialization dataWithJSONObject:@{@"paragraphs":paragraphs} options:0 error:NULL];
            cr_insert((uintptr_t)(__bridge void *)self.owner,[[NSString alloc] initWithData:data encoding:NSUTF8StringEncoding].UTF8String);
            return;
        }
        NSMutableAttributedString *value=[[self.owner attributedDocument:@{@"paragraphs":paragraphs}] mutableCopy];
        // Inline rich paste belongs to the destination paragraph, not the
        // source block's round-trip metadata.
        if (!opaque && paragraphs.count==1 && value.length) [value addAttribute:CRParagraph value:CRParagraphAt(self.owner,self.selectedRange.location) range:NSMakeRange(0,value.length)];
        [self.owner replace:self.selectedRange with:value selection:NSMakeRange(self.selectedRange.location+value.length,0)];
        return;
    }
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
    e.text.drawsBackground=NO; // CRTextView.drawRect paints canvas and chips itself
    e.draftGroup=NSMakeRange(NSNotFound,0);
    e.lastCaret=NSNotFound;
    [e.text registerForDraggedTypes:@[NSFilenamesPboardType,NSPasteboardTypeFileURL]];
    e.mathCache=[NSMutableDictionary new]; e.imageCache=[NSMutableDictionary new]; e.imagePending=[NSMutableDictionary new]; e.imageAttachments=[NSMutableDictionary new];
    e.text.typingAttributes=@{CRParagraph:CRBody()}; e.scroll.documentView=e.text; [w.contentView addSubview:e.scroll positioned:NSWindowAbove relativeTo:nil];
    [e restyle];
    return (uintptr_t)(__bridge_retained void *)e;
}
void cr_close(uintptr_t ptr) { CREditor *e=(__bridge_transfer CREditor *)(void *)ptr; e.callback=0; e.text.delegate=nil; [e.scroll removeFromSuperview]; [e.undo removeAllActions]; }
void cr_load(uintptr_t ptr,const char *json) { CREditor *e=CRE(ptr); e.suppress=YES; NSAttributedString *value=[e attributedDocument:CRJSON(json)]; [e.text.textStorage setAttributedString:value]; e.text.selectedRange=NSMakeRange(0,0); e.text.typingAttributes=@{CRParagraph: value.length ? [value attribute:CRParagraph atIndex:0 effectiveRange:NULL] : e.emptyParagraph ?: CRBody()}; [e.undo removeAllActions]; e.suppress=NO; [e restyle]; }
char *cr_snapshot(uintptr_t ptr,int *marked) { CREditor *e=CRE(ptr); *marked=e.text.hasMarkedText; if (*marked) return NULL; NSData *data=[NSJSONSerialization dataWithJSONObject:[e document] options:0 error:NULL]; if (!data) return NULL; return strdup([[NSString alloc] initWithData:data encoding:NSUTF8StringEncoding].UTF8String); }
void cr_format(uintptr_t ptr,const char *command) { [CRE(ptr) format:[NSString stringWithUTF8String:command]]; }
void cr_insert(uintptr_t ptr,const char *json) {
    CREditor *e=CRE(ptr); NSRange selection=e.text.selectedRange;
    if (!e.text.editable || e.text.hasMarkedText || ![e allowRange:selection]) return;
    NSTextTableBlock *destination=CRCellAt(e,selection.location);
    if (destination) selection=NSMakeRange(NSMaxRange(CRTableRange(e,destination.table)),0);
    NSMutableDictionary *doc=[CRJSON(json) mutableCopy]; NSMutableArray *paragraphs=[NSMutableArray array];
    for (NSDictionary *original in doc[@"paragraphs"]) { NSMutableDictionary *p=[original mutableCopy]; [p removeObjectForKey:@"id"]; [p removeObjectForKey:@"nextId"]; [paragraphs addObject:p]; } doc[@"paragraphs"]=paragraphs;
    NSDictionary *ending=e.emptyParagraph;
    NSMutableAttributedString *insert=[[e attributedDocument:doc] mutableCopy];
    e.emptyParagraph=ending;
    NSDictionary *attrs=[e visual:@{CRParagraph:CRBody()} paragraph:CRBody()];
    if (selection.location && [e.text.string characterAtIndex:selection.location-1]!='\n') [insert insertAttributedString:[[NSAttributedString alloc] initWithString:@"\n" attributes:attrs] atIndex:0];
    [insert appendAttributedString:[[NSAttributedString alloc] initWithString:@"\n" attributes:attrs]];
    __block NSUInteger tableStart=NSNotFound;
    [insert enumerateAttribute:CRTableCell inRange:NSMakeRange(0,insert.length) options:0 usingBlock:^(id cell, NSRange range, BOOL *stop) { if (cell) { tableStart=range.location; *stop=YES; } }];
    e.suppress=YES;
    [e replace:selection with:insert selection:NSMakeRange(selection.location+(tableStart==NSNotFound ? insert.length : tableStart),0)];
    e.text.typingAttributes=attrs; e.emptyParagraph=CRBody(); e.suppress=NO;
    [e updateTypingParagraph]; [e notify];
}
// CRUpdateInset caps the reading measure: past ~47 CJK chars per line the eye
// loses the return sweep, so on wide panes the text column centers itself
// (Typora-style) instead of stretching edge to edge.
static void CRUpdateInset(CREditor *e) {
    CGFloat maxW=760*e.zoom;
    CGFloat width=e.scroll.contentSize.width;
    CGFloat x=MAX(36*e.zoom,(width-maxW)/2);
    NSSize inset=NSMakeSize(x,22*e.zoom);
    if (!NSEqualSizes(e.text.textContainerInset,inset)) e.text.textContainerInset=inset;
}
void cr_frame(uintptr_t ptr,double x,double y,double w,double h) { CREditor *e=CRE(ptr); NSView *parent=e.scroll.superview; CGFloat top=parent.isFlipped ? y : parent.bounds.size.height-y-h; NSRect frame=NSMakeRect(x,top,MAX(0,w),MAX(0,h)); if (!NSEqualRects(frame,e.scroll.frame)) { e.scroll.frame=frame; CRUpdateInset(e); } }
void cr_visible(uintptr_t ptr,int visible) { CREditor *e=CRE(ptr); if (!visible && e.window.firstResponder==e.text) cr_focus(ptr,0); e.scroll.hidden=!visible; }
void cr_base_dir(uintptr_t ptr,const char *path) { CRE(ptr).baseDir=[NSString stringWithUTF8String:path]; }
void cr_style(uintptr_t ptr,double zoom,int dark,int editable) {
    CREditor *e=CRE(ptr); zoom=MAX(0.5,MIN(3,zoom));
    if (e.zoom==zoom && e.dark==dark && e.text.editable==editable) return;
    BOOL figures=e.zoom!=zoom || e.dark!=dark;
    e.zoom=zoom; e.dark=dark; e.text.editable=editable;
    CRUpdateInset(e);
    if (figures && !e.text.hasMarkedText) {
        // Attachments (formula images, sizing) are baked into the storage, so
        // a zoom or theme change rebuilds the document in place.
        NSDictionary *doc=[e document];
        NSRange selection=e.text.selectedRange;
        BOOL wasSuppress=e.suppress; e.suppress=YES;
        [e.text.textStorage setAttributedString:[e attributedDocument:doc]];
        if (selection.location>e.text.string.length) selection=NSMakeRange(e.text.string.length,0);
        e.text.selectedRange=selection;
        e.suppress=wasSuppress;
    }
    [e restyle];
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
uintptr_t cr_test_window(void) { NSWindow *window=[[NSWindow alloc] initWithContentRect:NSMakeRect(0,0,720,500) styleMask:NSWindowStyleMaskTitled backing:NSBackingStoreBuffered defer:NO]; window.releasedWhenClosed=NO; return (uintptr_t)(__bridge_retained void *)window; }
void cr_test_close_window(uintptr_t ptr) { NSWindow *window=(__bridge_transfer NSWindow *)(void *)ptr; [window close]; }
void cr_test_select(uintptr_t ptr,int start,int length) { CRE(ptr).text.selectedRange=NSMakeRange(start,length); }
void cr_test_selection(uintptr_t ptr,int *start,int *length) { NSRange r=CRE(ptr).text.selectedRange; *start=(int)r.location; *length=(int)r.length; }
void cr_test_action(uintptr_t ptr,const char *action,const char *text) {
    CREditor *e=CRE(ptr); NSString *a=[NSString stringWithUTF8String:action], *s=[NSString stringWithUTF8String:text];
    if ([a isEqual:@"undo"]) { [e.text breakUndoCoalescing]; [e.undo undo]; }
    else if ([a isEqual:@"newline"]) [e.text insertNewline:nil];
    else if ([a isEqual:@"reveal"]) [e revealSourceAt:e.text.selectedRange.location];
    else if ([a isEqual:@"blur"]) [e textDidEndEditing:[NSNotification notificationWithName:NSTextDidEndEditingNotification object:e.text]];
    else if ([a isEqual:@"linebreak"]) [e.text insertLineBreak:nil];
    else if ([a isEqual:@"tab"]) [e.text insertTab:nil];
    else if ([a isEqual:@"backtab"]) [e.text insertBacktab:nil];
    else if ([a isEqual:@"redo"]) [e.undo redo];
    else if ([a isEqual:@"paste"]) { [e.text breakUndoCoalescing]; [NSPasteboard.generalPasteboard clearContents]; [NSPasteboard.generalPasteboard setString:s forType:NSPasteboardTypeString]; [e.text paste:nil]; [e.text breakUndoCoalescing]; }
    else if ([a isEqual:@"copy"]) [e.text copy:nil];
    else if ([a isEqual:@"pasteboard"]) [e.text paste:nil];
    else if ([a isEqual:@"underlinePaste"]) { NSMutableAttributedString *rich=[[NSMutableAttributedString alloc] initWithString:@"underlined" attributes:@{NSUnderlineStyleAttributeName:@1}]; NSData *rtf=[rich RTFFromRange:NSMakeRange(0,rich.length) documentAttributes:@{}]; [NSPasteboard.generalPasteboard clearContents]; [NSPasteboard.generalPasteboard setData:rtf forType:NSPasteboardTypeRTF]; [e.text paste:nil]; }
    else if ([a isEqual:@"unsupportedPaste"]) { NSMutableAttributedString *rich=[[NSMutableAttributedString alloc] initWithString:@"superscript" attributes:@{NSSuperscriptAttributeName:@1}]; NSData *rtf=[rich RTFFromRange:NSMakeRange(0,rich.length) documentAttributes:@{}]; [NSPasteboard.generalPasteboard clearContents]; [NSPasteboard.generalPasteboard setData:rtf forType:NSPasteboardTypeRTF]; [e.text paste:nil]; }
    else if ([a isEqual:@"backspace"]) [e.text deleteBackward:nil];
    else if ([a isEqual:@"pngPaste"]) { NSBitmapImageRep *rep=[[NSBitmapImageRep alloc] initWithBitmapDataPlanes:NULL pixelsWide:4 pixelsHigh:4 bitsPerSample:8 samplesPerPixel:4 hasAlpha:YES isPlanar:NO colorSpaceName:NSCalibratedRGBColorSpace bytesPerRow:0 bitsPerPixel:0]; [NSPasteboard.generalPasteboard clearContents]; [NSPasteboard.generalPasteboard setData:[rep representationUsingType:NSBitmapImageFileTypePNG properties:@{}] forType:NSPasteboardTypePNG]; [e.text paste:nil]; }
    else if ([a isEqual:@"htmlPaste"]) { [NSPasteboard.generalPasteboard clearContents]; [NSPasteboard.generalPasteboard setString:s forType:NSPasteboardTypeHTML]; [e.text paste:nil]; }
    else if ([a isEqual:@"filePaste"]) { [NSPasteboard.generalPasteboard clearContents]; [NSPasteboard.generalPasteboard setPropertyList:[s componentsSeparatedByString:@"|"] forType:NSFilenamesPboardType]; [e.text paste:nil]; }
    else if ([a isEqual:@"mark"]) [e.text setMarkedText:s selectedRange:NSMakeRange(s.length,0) replacementRange:e.text.selectedRange];
    else if ([a isEqual:@"commit"]) [e.text insertText:s replacementRange:e.text.markedRange];
    // Real IMEs commit with NSNotFound, letting the view replace marked text.
    else if ([a isEqual:@"commitIME"]) [e.text insertText:s replacementRange:NSMakeRange(NSNotFound,0)];
    else [e.text insertText:s replacementRange:e.text.selectedRange];
}
int cr_test_traits(uintptr_t ptr,const char *trait) { CREditor *e=CRE(ptr); NSString *key=[NSString stringWithUTF8String:trait]; __block BOOL found=NO; [e.text.textStorage enumerateAttributesInRange:NSMakeRange(0,e.text.string.length) options:0 usingBlock:^(NSDictionary *attrs, NSRange r, BOOL *stop) { if ([key isEqual:@"bold"]) found |= ([[NSFontManager sharedFontManager] traitsOfFont:attrs[NSFontAttributeName]] & NSBoldFontMask)!=0; else if ([key isEqual:@"underline"]) found |= [attrs[NSUnderlineStyleAttributeName] integerValue]!=0; else if ([key isEqual:@"highlight"]) found |= attrs[CRHighlight]!=nil && attrs[NSBackgroundColorAttributeName]!=nil; else if ([key isEqual:@"sup"]) found |= [attrs[NSSuperscriptAttributeName] integerValue]>0; else if ([key isEqual:@"sub"]) found |= [attrs[NSSuperscriptAttributeName] integerValue]<0; else if ([key hasPrefix:@"hl:"]) found |= [attrs[CRHl] isEqual:[key substringFromIndex:3]]; else found |= CRKind(attrs[CRParagraph],key); }];
    if ([key isEqual:@"scrolled"]) return e.text.visibleRect.origin.y > 0.5;
    if ([key isEqual:@"findbar"]) return e.text.usesFindBar ? 1 : 0;
    if ([key hasPrefix:@"alphaAt:"]) { NSInteger index=[[key substringFromIndex:8] integerValue]; if (index>=0 && index<e.text.textStorage.length) { NSColor *color=[e.text.textStorage attribute:NSForegroundColorAttributeName atIndex:index effectiveRange:NULL]; return color && color.alphaComponent<0.99; } return 0; }
    return found; }
int cr_test_checkbox(uintptr_t ptr) { CREditor *e=CRE(ptr); for (NSValue *v in CRRanges(e.text.string)) if (CRKind(CRParagraphAt(e,v.rangeValue.location),@"checklist")) { NSRect r=[e markerRect:v.rangeValue]; NSPoint point=[e.text convertPoint:NSMakePoint(NSMidX(r),NSMidY(r)) toView:nil]; NSEvent *event=[NSEvent mouseEventWithType:NSEventTypeLeftMouseDown location:point modifierFlags:0 timestamp:0 windowNumber:e.window.windowNumber context:nil eventNumber:0 clickCount:1 pressure:1]; [e.text mouseDown:event]; return 1; } return 0; }
void cr_test_opaque(uintptr_t ptr,int whole) { CREditor *e=CRE(ptr); for (NSValue *v in CRRanges(e.text.string)) if (CRKind(CRParagraphAt(e,v.rangeValue.location),@"opaque")) { NSRange r=CRContent(e.text.string,v.rangeValue); if (!whole) r=NSMakeRange(r.location+1,1); e.text.selectedRange=r; return; } }
int cr_test_screenshot(uintptr_t ptr,const char *path) { CREditor *e=CRE(ptr); [e.scroll layoutSubtreeIfNeeded]; [e.text.layoutManager ensureLayoutForTextContainer:e.text.textContainer]; NSBitmapImageRep *rep=[e.scroll bitmapImageRepForCachingDisplayInRect:e.scroll.bounds]; [e.scroll cacheDisplayInRect:e.scroll.bounds toBitmapImageRep:rep]; NSData *png=[rep representationUsingType:NSBitmapImageFileTypePNG properties:@{}]; return [png writeToFile:[NSString stringWithUTF8String:path] atomically:YES]; }
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
    if (_text.typingAttributes[CRCode] || _text.typingAttributes[CRMathText]) return NO;
    if (caret && caret<=_text.textStorage.length) {
        NSDictionary *before=[_text.textStorage attributesAtIndex:caret-1 effectiveRange:NULL];
        if (before[CRCode] || before[CRMathText]) return NO;
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
- (BOOL)autoPair:(id)value range:(NSRange)range {
    NSString *s=[value isKindOfClass:NSAttributedString.class] ? [value string] : value;
    if (_suppress || !_text.editable || _text.hasMarkedText) return NO;
    if (s.length!=1) return NO;
    unichar c=[s characterAtIndex:0];
    NSString *closer=@{@"(":@")",@"[":@"]",@"{":@"}",@"\"":@"\"",@"'":@"'"}[s];
    BOOL wrapOnly=!closer && [@"`*_~" rangeOfString:s].location!=NSNotFound;
    BOOL isCloser=[@")]}\"'" rangeOfString:s].location!=NSNotFound;
    if (!closer && !wrapOnly && !isCloser) return NO;
    NSDictionary *p=CRParagraphAt(self,range.location);
    if (CRKind(p,@"code") || CRKind(p,@"source") || CRKind(p,@"math") || CRKind(p,@"opaque")) return NO;
    if (CRCellAt(self,range.location)) return NO;
    if (_text.typingAttributes[CRCode] || _text.typingAttributes[CRMathText]) return NO;
    if (range.location && range.location<=_text.textStorage.length) {
        NSDictionary *before=[_text.textStorage attributesAtIndex:range.location-1 effectiveRange:NULL];
        if (before[CRCode] || before[CRMathText]) return NO;
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
        NSMutableAttributedString *after=[_text.textStorage mutableCopy];
        NSDictionary *openAttrs=[after attributesAtIndex:range.location effectiveRange:NULL];
        NSDictionary *closeAttrs=[after attributesAtIndex:NSMaxRange(range)-1 effectiveRange:NULL];
        [after insertAttributedString:[[NSAttributedString alloc] initWithString:closer ?: s attributes:closeAttrs] atIndex:NSMaxRange(range)];
        [after insertAttributedString:[[NSAttributedString alloc] initWithString:s attributes:openAttrs] atIndex:range.location];
        [self replace:NSMakeRange(0,_text.string.length) with:after selection:NSMakeRange(range.location+1,range.length)];
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
void cr_reveal_heading(uintptr_t ptr, int index) { [CRE(ptr) revealHeading:(NSInteger)index]; }
void cr_print(uintptr_t ptr) { [CRE(ptr) printDocument]; }

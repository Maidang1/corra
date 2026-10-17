// A "[TOC]" paragraph renders the document's headings as a table of contents.
// The marker itself is the only text the document keeps, so the figure is drawn
// as one attachment and refreshed whenever the headings change: adding, editing
// or removing a heading redraws the list in place, and picking an entry scrolls
// to the heading it names.

// CRTOCLine is the height of one entry, fixed so a click can be mapped to an
// entry without measuring the drawn text.
static CGFloat CRTOCLine(CGFloat zoom) { return 22 * zoom; }

// CRTOCStep is the indent one outline level adds.
static CGFloat CRTOCStep(CGFloat zoom) { return 16 * zoom; }

// CRTOCWidth is the column the list is drawn in, which is the same column a
// figure of the document has to fit.
static CGFloat CRTOCWidth(CREditor *e) { return MAX(60, e.text.textContainer.size.width - 8 * e.zoom); }

// CRTOCEntry builds one line of the list.
static NSDictionary *CRTOCEntry(NSInteger level, NSString *text) {
    return @{@"level": @(MAX(1, level)), @"text": text ?: @""};
}

// CRTOCSignature is the shape of the list, so a redraw only happens when a
// heading was added, removed, renamed or moved.
static NSString *CRTOCSignature(NSArray *entries) {
    NSMutableArray *lines = [NSMutableArray array];
    for (NSDictionary *entry in entries) [lines addObject:[NSString stringWithFormat:@"%@|%@", entry[@"level"], entry[@"text"]]];
    return [lines componentsJoinedByString:@"\n"];
}

// tocFigureRanges lists the [TOC] paragraphs whose figure is drawn, which is the
// only case the list can be replaced in place.
- (NSArray *)tocFigureRanges {
    NSMutableArray *figures = [NSMutableArray array];
    for (NSValue *value in CRRanges(_text.string)) {
        NSRange range = value.rangeValue;
        NSRange content = CRContent(_text.string, range);
        if (!CRKind(CRParagraphAt(self, range.location), @"toc")) continue;
        if (content.length != 1 || ![[_text.string substringWithRange:content] isEqual:@"\uFFFC"]) continue;
        [figures addObject:value];
    }
    return figures;
}

// tocEntriesFromText reads the headings out of the text storage, which is the
// truth even while a document is being loaded or a paragraph is being replaced.
- (NSArray *)tocEntriesFromText {
    NSMutableArray *entries = [NSMutableArray array];
    for (NSValue *value in CRRanges(_text.string)) {
        NSRange range = value.rangeValue;
        NSDictionary *p = CRParagraphAt(self, range.location);
        if (!CRKind(p, @"heading")) continue;
        NSString *text = [[_text.string substringWithRange:CRContent(_text.string, range)] stringByTrimmingCharactersInSet:NSCharacterSet.whitespaceAndNewlineCharacterSet];
        if (!text.length) continue;
        [entries addObject:CRTOCEntry([p[@"level"] integerValue], text)];
    }
    return entries;
}

// tocImage draws the list. Every entry is exactly one line tall, indented by the
// level it sits at, so the picture and the hit test agree. A document with no
// headings gets a one point picture: the marker stays, the list is empty.
- (NSImage *)tocImage:(NSArray *)entries width:(CGFloat)width {
    CGFloat line = CRTOCLine(_zoom), pad = 8 * _zoom;
    NSSize size = NSMakeSize(MAX(1, width), MAX(1, line * entries.count));
    if (!entries.count) {
        return [NSImage imageWithSize:size flipped:NO drawingHandler:^BOOL(NSRect rect) { return YES; }];
    }
    NSMutableAttributedString *text = [NSMutableAttributedString new];
    for (NSUInteger i = 0; i < entries.count; i++) {
        NSDictionary *entry = entries[i];
        NSInteger level = [entry[@"level"] integerValue];
        NSMutableParagraphStyle *style = [NSMutableParagraphStyle new];
        style.minimumLineHeight = line;
        style.maximumLineHeight = line;
        style.headIndent = pad + (level - 1) * CRTOCStep(_zoom);
        style.firstLineHeadIndent = style.headIndent;
        NSDictionary *attributes = @{
            NSFontAttributeName: [NSFont systemFontOfSize:13 * _zoom weight:level == 1 ? NSFontWeightSemibold : NSFontWeightRegular],
            NSForegroundColorAttributeName: _dark ? [NSColor colorWithWhite:0.86 alpha:1] : [NSColor colorWithWhite:0.22 alpha:1],
            NSParagraphStyleAttributeName: style,
        };
        [text appendAttributedString:[[NSAttributedString alloc] initWithString:entry[@"text"] ?: @"" attributes:attributes]];
        // No trailing newline: it would draw a line the list does not have.
        if (i + 1 < entries.count) [text appendAttributedString:[[NSAttributedString alloc] initWithString:@"\n" attributes:attributes]];
    }
    return [NSImage imageWithSize:size flipped:YES drawingHandler:^BOOL(NSRect rect) {
        [text drawWithRect:NSMakeRect(0, 0, size.width, size.height) options:NSStringDrawingUsesLineFragmentOrigin];
        return YES;
    }];
}

// tocFigure draws the list of the document's headings. While a document is being
// loaded the text storage is still empty, so the figure starts as a placeholder
// that refreshTOC replaces as soon as the text is in place.
- (NSAttributedString *)tocFigure:(NSDictionary *)p {
    NSImage *image = [self tocImage:[self tocEntriesFromText] width:CRTOCWidth(self)];
    NSTextAttachment *attachment = [NSTextAttachment new];
    attachment.image = image;
    attachment.bounds = NSMakeRect(0, 0, image.size.width, image.size.height);
    // The figure carries the paragraph it belongs to, like every other figure,
    // or the editor cannot tell which marker it is looking at.
    NSAttributedString *figure = [NSAttributedString attributedStringWithAttachment:attachment attributes:[self visual:@{CRParagraph:p} paragraph:p]];
    return figure;
}

// refreshTOC redraws every [TOC] figure when the headings changed. Only the
// attachment is replaced, never the text, so the marker keeps its place and its
// undo history.
- (void)refreshTOC {
    if (_suppress || _text.hasMarkedText) return;
    // Without a [TOC] block there is nothing to redraw; forget the signature
    // so one inserted later renders fresh.
    if (![self tocFigureRanges].count) { _tocSignature = nil; return; }
    NSArray *entries = [self tocEntriesFromText];
    NSString *signature = CRTOCSignature(entries);
    if ([signature isEqual:_tocSignature]) return;
    _tocSignature = signature;
    NSImage *image = [self tocImage:entries width:CRTOCWidth(self)];
    BOOL previous = _suppress;
    _suppress = YES;
    for (NSValue *value in [self tocFigureRanges]) {
        NSRange content = CRContent(_text.string, value.rangeValue);
        NSTextAttachment *attachment = [NSTextAttachment new];
        attachment.image = image;
        attachment.bounds = NSMakeRect(0, 0, image.size.width, image.size.height);
        [_text.textStorage addAttribute:NSAttachmentAttributeName value:attachment range:content];
    }
    _suppress = previous;
    [_text.layoutManager invalidateLayoutForCharacterRange:NSMakeRange(0, _text.textStorage.length) actualCharacterRange:NULL];
    [_text setNeedsDisplay:YES];
}

// tocEntryAtPoint maps a click inside a [TOC] figure to the entry it landed on,
// or -1 when the click fell outside the list.
- (NSInteger)tocEntryAtPoint:(NSPoint)point range:(NSRange)range {
    NSPoint container = NSMakePoint(point.x - _text.textContainerOrigin.x, point.y - _text.textContainerOrigin.y);
    NSRect rect = [_text.layoutManager boundingRectForGlyphRange:NSMakeRange([_text.layoutManager glyphIndexForCharacterAtIndex:range.location], 1) inTextContainer:_text.textContainer];
    if (!NSPointInRect(container, rect)) return -1;
    NSInteger entry = (NSInteger)((container.y - rect.origin.y) / CRTOCLine(_zoom));
    if (entry < 0 || (NSUInteger)entry >= [self tocEntriesFromText].count) return -1;
    return entry;
}

// jumpToTOCEntry scrolls the heading an entry names into view and puts the caret
// on it, so the reader lands where they picked.
- (void)jumpToTOCEntry:(NSInteger)entry {
    NSInteger seen = 0;
    for (NSValue *value in CRRanges(_text.string)) {
        NSRange range = value.rangeValue;
        if (!CRKind(CRParagraphAt(self, range.location), @"heading")) continue;
        if (seen++ == entry) {
            [_text scrollRangeToVisible:range];
            if (_text.editable) _text.selectedRange = NSMakeRange(range.location, 0);
            return;
        }
    }
}

// CRSlug is the GitHub/Typora heading anchor: lowercase, punctuation
// dropped (except - and _), whitespace runs turned into hyphens.
static NSString *CRSlug(NSString *text) {
    NSMutableString *slug=[NSMutableString string];
    NSString *lower=text.lowercaseString;
    NSCharacterSet *space=NSCharacterSet.whitespaceCharacterSet, *keep=NSCharacterSet.alphanumericCharacterSet;
    for (NSUInteger i=0;i<lower.length;) {
        NSRange composed=[lower rangeOfComposedCharacterSequenceAtIndex:i];
        unichar c=[lower characterAtIndex:i];
        if ([space characterIsMember:c]) [slug appendString:@"-"];
        else if (c=='-' || c=='_' || [keep characterIsMember:c] || composed.length>1) [slug appendString:[lower substringWithRange:composed]];
        i=NSMaxRange(composed);
    }
    return slug;
}

// jumpToAnchor scrolls to the heading a "#fragment" link names and returns
// NO when no heading matches.
- (BOOL)jumpToAnchor:(NSString *)fragment {
    NSString *wanted=CRSlug([fragment stringByRemovingPercentEncoding] ?: fragment);
    for (NSValue *value in CRRanges(_text.string)) {
        NSRange range = value.rangeValue;
        if (!CRKind(CRParagraphAt(self, range.location), @"heading")) continue;
        NSString *text=[[_text.string substringWithRange:CRContent(_text.string,range)] stringByTrimmingCharactersInSet:NSCharacterSet.whitespaceAndNewlineCharacterSet];
        if (![CRSlug(text) isEqual:wanted] && ![CRSlug(text) isEqual:CRSlug(fragment)]) continue;
        [_text scrollRangeToVisible:range];
        if (_text.editable) _text.selectedRange = NSMakeRange(range.location, 0);
        return YES;
    }
    return NO;
}

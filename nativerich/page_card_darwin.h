// Page cards: a paragraph holding only "![Title](note.md)" embeds another note
// as a card (Notion's "link to page" block). The card shows the note's live
// title and its first lines, and opens the note on click. Markdown's "!"
// already means "embed this resource", so other tools read it as a reference
// to the note. Included inside @implementation CREditor.

#pragma once

// CRPageTarget is the note a page paragraph points at, as written.
static NSString *CRPageTarget(NSDictionary *p) {
    NSString *raw = [p[@"raw"] isKindOfClass:NSString.class] ? p[@"raw"] : @"";
    NSString *url = CRImageRawURL(raw);
    return CRLocalDocument(url) ? url : nil;
}

// CRPageAlt is the card's stored title: the image's alt text.
static NSString *CRPageAlt(NSDictionary *p) {
    NSString *raw = [p[@"raw"] isKindOfClass:NSString.class] ? p[@"raw"] : @"";
    NSRange open = [raw rangeOfString:@"!["], close = [raw rangeOfString:@"]("];
    if (open.location == NSNotFound || close.location == NSNotFound || close.location < NSMaxRange(open)) return @"";
    NSString *alt = [raw substringWithRange:NSMakeRange(NSMaxRange(open), close.location - NSMaxRange(open))];
    return [alt stringByReplacingOccurrencesOfString:@"\\" withString:@""];
}

// CRPageMarkdown writes the card for a note: brackets in the title are
// escaped and a path with spaces takes the <...> form.
static NSString *CRPageMarkdown(NSString *title, NSString *path) {
    NSString *alt = [[[title stringByReplacingOccurrencesOfString:@"\\" withString:@"\\\\"]
        stringByReplacingOccurrencesOfString:@"[" withString:@"\\["] stringByReplacingOccurrencesOfString:@"]" withString:@"\\]"];
    NSString *destination = [path rangeOfCharacterFromSet:[NSCharacterSet characterSetWithCharactersInString:@" ()<>"]].location == NSNotFound
        ? path : [NSString stringWithFormat:@"<%@>", [[path stringByReplacingOccurrencesOfString:@"<" withString:@"%3C"] stringByReplacingOccurrencesOfString:@">" withString:@"%3E"]];
    return [NSString stringWithFormat:@"![%@](%@)", alt, destination];
}

static CGFloat CRPageCardHeight(CGFloat zoom) { return 64 * zoom; }

// pageInfo reads the note a card points at: {title, excerpt, missing}.
- (NSDictionary *)pageInfo:(NSString *)target {
    NSString *path = [[target componentsSeparatedByString:@"#"].firstObject stringByRemovingPercentEncoding] ?: target;
    if (![path hasPrefix:@"/"] && _baseDir.length) path = [_baseDir stringByAppendingPathComponent:path];
    char *json = cr_page_info(path.UTF8String);
    if (!json) return @{@"missing": @YES};
    NSData *data = [[NSString stringWithUTF8String:json] dataUsingEncoding:NSUTF8StringEncoding];
    free(json);
    NSDictionary *info = data ? [NSJSONSerialization JSONObjectWithData:data options:0 error:NULL] : nil;
    return [info isKindOfClass:NSDictionary.class] ? info : @{@"missing": @YES};
}

// pageCardImage draws the card: a hairline-bordered panel with a page glyph,
// the note's title and its first lines, or a "missing" line when the note is
// gone.
- (NSImage *)pageCardImage:(NSDictionary *)p width:(CGFloat)width {
    NSString *target = CRPageTarget(p) ?: @"";
    NSDictionary *info = [self pageInfo:target];
    BOOL missing = [info[@"missing"] boolValue];
    NSString *title = [info[@"title"] length] ? info[@"title"] : CRPageAlt(p);
    if (!title.length) title = target.lastPathComponent.stringByDeletingPathExtension;
    NSString *detail = missing ? [NSString stringWithFormat:@"找不到文档 %@", target] : ([info[@"excerpt"] length] ? info[@"excerpt"] : target);
    CGFloat zoom = _zoom, height = CRPageCardHeight(zoom);
    BOOL dark = _dark;
    NSSize size = NSMakeSize(MAX(120, width), height);
    return [NSImage imageWithSize:size flipped:YES drawingHandler:^BOOL(NSRect rect) {
        NSRect card = NSInsetRect(NSMakeRect(0, 0, size.width, size.height), 0.5, 0.5);
        NSBezierPath *shape = [NSBezierPath bezierPathWithRoundedRect:card xRadius:8 * zoom yRadius:8 * zoom];
        [(dark ? [NSColor colorWithWhite:1 alpha:0.04] : [NSColor colorWithWhite:0 alpha:0.018]) setFill];
        [shape fill];
        [(dark ? [NSColor colorWithWhite:1 alpha:0.14] : [NSColor colorWithWhite:0 alpha:0.12]) setStroke];
        shape.lineWidth = 1;
        [shape stroke];
        NSColor *ink = dark ? [NSColor colorWithWhite:0.9 alpha:1] : [NSColor colorWithWhite:0.16 alpha:1];
        NSColor *muted = missing ? [NSColor colorWithSRGBRed:0.83 green:0.30 blue:0.28 alpha:1] : CRMutedColor(dark);
        CGFloat pad = 14 * zoom, icon = 22 * zoom;
        NSImage *glyph = CRTintedSymbol(missing ? @"doc.questionmark" : @"doc.text", 17 * zoom, NSFontWeightRegular, missing ? muted : CRMutedColor(dark));
        if (glyph) {
            NSSize g = glyph.size;
            [glyph drawInRect:NSMakeRect(pad + (icon - g.width) / 2, (height - g.height) / 2, g.width, g.height)
                     fromRect:NSZeroRect operation:NSCompositingOperationSourceOver fraction:1 respectFlipped:YES hints:nil];
        }
        CGFloat x = pad + icon + 10 * zoom, textWidth = size.width - x - pad;
        NSMutableParagraphStyle *clip = [NSMutableParagraphStyle new];
        clip.lineBreakMode = NSLineBreakByTruncatingTail;
        NSDictionary *titleAttrs = @{NSFontAttributeName: [NSFont systemFontOfSize:15 * zoom weight:NSFontWeightSemibold], NSForegroundColorAttributeName: ink, NSParagraphStyleAttributeName: clip};
        NSDictionary *detailAttrs = @{NSFontAttributeName: [NSFont systemFontOfSize:12.5 * zoom], NSForegroundColorAttributeName: muted, NSParagraphStyleAttributeName: clip};
        [title drawWithRect:NSMakeRect(x, 13 * zoom, textWidth, 20 * zoom) options:NSStringDrawingUsesLineFragmentOrigin attributes:titleAttrs];
        [detail drawWithRect:NSMakeRect(x, 35 * zoom, textWidth, 17 * zoom) options:NSStringDrawingUsesLineFragmentOrigin attributes:detailAttrs];
        return YES;
    }];
}

// pageFigure is the card as one attachment carrying its paragraph, like every
// other figure.
- (NSAttributedString *)pageFigure:(NSDictionary *)p {
    if (!CRPageTarget(p)) return [self sourceText:p];
    NSImage *image = [self pageCardImage:p width:MAX(60, _text.textContainer.size.width - 8 * _zoom)];
    NSTextAttachment *attachment = [NSTextAttachment new];
    attachment.image = image;
    attachment.bounds = NSMakeRect(0, 0, image.size.width, image.size.height);
    return [NSAttributedString attributedStringWithAttachment:attachment attributes:[self visual:@{CRParagraph: p} paragraph:p]];
}

// insertPageCard turns the paragraph holding only the typed "@query" into a
// card for the note. NO when the paragraph holds other text: the pick then
// stays an inline mention.
- (BOOL)insertPageCard:(NSString *)title path:(NSString *)path replacing:(NSRange)range {
    NSRange paragraph = [_text.string paragraphRangeForRange:NSMakeRange(range.location, 0)];
    if (!NSEqualRanges(CRContent(_text.string, paragraph), range)) return NO;
    if (!CRKind(CRParagraphAt(self, range.location), @"body")) return NO;
    [self beginStructuralCommand];
    [_undo beginUndoGrouping];
    BOOL rendered = [self renderSource:CRPageMarkdown(title, path) inRange:paragraph];
    [_undo endUndoGrouping];
    return rendered;
}

// beginPageReference types the "@" that opens the note picker (the slash
// menu's 提及页面 / 页面卡片). For a card the pick then replaces the paragraph.
- (void)beginPageReference:(BOOL)card {
    if (!_text.editable || _text.hasMarkedText) return;
    NSRange caret = _text.selectedRange;
    NSUInteger at = caret.location;
    NSMutableDictionary *attrs = [_text.typingAttributes mutableCopy];
    for (NSString *key in @[NSLinkAttributeName, CRLinkTitle, CRMention, CRCode, CRInlineSource]) [attrs removeObjectForKey:key];
    NSString *text = @"@";
    // "@" right after a word reads as an address, not a mention.
    if (at > 0 && at <= _text.string.length) {
        unichar before = [_text.string characterAtIndex:at - 1];
        if (before < 0x80 && [[NSCharacterSet alphanumericCharacterSet] characterIsMember:before]) { text = @" @"; at++; }
    }
    NSDictionary *paragraph = attrs[CRParagraph] ?: CRBody();
    [self replace:caret with:[[NSAttributedString alloc] initWithString:text attributes:[self visual:attrs paragraph:paragraph]]
        selection:NSMakeRange(caret.location + text.length, 0)];
    _linkCard = card;
    _linkCardAt = at;
    [self linkMenuQueryChanged];
}

// pageCardState lists the cards for the Go test harness.
- (NSDictionary *)pageCardState {
    NSMutableArray *cards = [NSMutableArray array];
    for (NSValue *value in CRRanges(_text.string)) {
        NSDictionary *p = CRParagraphAt(self, value.rangeValue.location);
        if (!CRKind(p, @"page")) continue;
        NSRange content = CRContent(_text.string, value.rangeValue);
        BOOL figure = content.length == 1 && [[_text.string substringWithRange:content] isEqual:@"\uFFFC"];
        NSDictionary *info = [self pageInfo:CRPageTarget(p) ?: @""];
        [cards addObject:@{@"target": CRPageTarget(p) ?: @"", @"figure": (figure ? @YES : @NO),
                           @"title": info[@"title"] ?: @"", @"excerpt": info[@"excerpt"] ?: @"",
                           @"missing": ([info[@"missing"] boolValue] ? @YES : @NO)}];
    }
    return @{@"cards": cards};
}

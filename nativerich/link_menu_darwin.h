// The link menu's editor half: typing "[[" opens note suggestions beside the
// caret so a note can be linked without typing its path. Included inside
// @implementation CREditor.

#pragma once

- (NSRange)linkMenuQueryRange {
    return [self linkMenuQueryRange:NULL];
}

// linkMenuQueryRange locates a "[[query" the caret sits at the end of. The
// query may contain spaces (note titles do); it stops at brackets or a line
// break. Failing that it locates an "@query" page mention, which stops at
// whitespace and must not follow a word ("a@b.com" is an address). Returns
// {NSNotFound, 0} elsewhere and inside protected blocks.
- (NSRange)linkMenuQueryRange:(BOOL *)mention {
    if (mention) *mention = NO;
    if (!_text.editable || _text.hasMarkedText || _suppress) return NSMakeRange(NSNotFound, 0);
    if (_text.selectedRange.length) return NSMakeRange(NSNotFound, 0);
    NSString *string = _text.string;
    NSUInteger caret = MIN(_text.selectedRange.location, string.length);
    NSRange paragraph = [string paragraphRangeForRange:NSMakeRange(caret, 0)];
    NSRange content = CRContent(string, paragraph);
    if (!content.length || caret < content.location || caret > NSMaxRange(content)) return NSMakeRange(NSNotFound, 0);
    NSDictionary *info = CRParagraphAt(self, content.location);
    if (CRKind(info, @"code") || CRKind(info, @"source") || CRKind(info, @"opaque") ||
        CRKind(info, @"math") || CRKind(info, @"image") || CRKind(info, @"diagram") ||
        CRKind(info, @"toc") || CRKind(info, @"page") || CRCellAt(self, content.location) || [info[@"footnoteId"] length]) {
        return NSMakeRange(NSNotFound, 0);
    }
    NSString *prefix = [string substringWithRange:NSMakeRange(content.location, caret - content.location)];
    NSRange opener = [prefix rangeOfString:@"[[" options:NSBackwardsSearch];
    if (opener.location != NSNotFound) {
        NSString *query = [prefix substringFromIndex:NSMaxRange(opener)];
        if (query.length <= 40 && [query rangeOfCharacterFromSet:[NSCharacterSet characterSetWithCharactersInString:@"[]\n\r"]].location == NSNotFound) {
            NSUInteger start = content.location + opener.location;
            return NSMakeRange(start, caret - start);
        }
    }
    NSRange at = [prefix rangeOfString:@"@" options:NSBackwardsSearch];
    if (at.location == NSNotFound) return NSMakeRange(NSNotFound, 0);
    NSString *query = [prefix substringFromIndex:NSMaxRange(at)];
    NSMutableCharacterSet *stops = [NSMutableCharacterSet whitespaceAndNewlineCharacterSet];
    [stops addCharactersInString:@"\u2028[]()#@"];
    if (query.length > 30 || [query rangeOfCharacterFromSet:stops].location != NSNotFound) return NSMakeRange(NSNotFound, 0);
    if (at.location > 0) {
        unichar before = [prefix characterAtIndex:at.location - 1];
        if (before < 0x80 && ([[NSCharacterSet alphanumericCharacterSet] characterIsMember:before] || strchr("._-/:+=&%~\\`", before))) {
            return NSMakeRange(NSNotFound, 0);
        }
    }
    NSUInteger start = content.location + at.location;
    // An "@" already inside a link, code or a mention is text.
    NSDictionary *attrs = [_text.textStorage attributesAtIndex:start effectiveRange:NULL];
    if (attrs[NSLinkAttributeName] || [attrs[CRCode] boolValue] || [attrs[CRMention] boolValue] || attrs[CRInlineSource]) return NSMakeRange(NSNotFound, 0);
    if (mention) *mention = YES;
    return NSMakeRange(start, caret - start);
}

- (NSString *)linkMenuFilterText {
    return _linkFilter ?: @"";
}

// linkMenuHeadingMode reports whether the query is a "#heading" search inside
// the open note rather than a note search.
- (BOOL)linkMenuHeadingMode {
    return [(_linkFilter ?: @"") hasPrefix:@"#"];
}

- (NSArray *)linkMenuItemsForFilter:(NSString *)filter {
    if ([filter hasPrefix:@"#"]) {
        NSString *query = [[filter substringFromIndex:1] lowercaseString];
        NSMutableArray *items = [NSMutableArray array];
        for (NSDictionary *entry in [self tocEntriesFromText]) {
            NSString *title = entry[@"text"] ?: @"";
            if (query.length && [title.lowercaseString rangeOfString:query].location == NSNotFound) continue;
            NSString *slug = CRSlug(title);
            if (!slug.length) continue;
            [items addObject:@{@"title": title, @"path": [@"#" stringByAppendingString:slug]}];
            if (items.count >= 24) break;
        }
        return items;
    }
    if (!_callback) return @[];
    char *json = cr_link_candidates(_callback, (char *)(filter ?: @"").UTF8String);
    if (!json) return @[];
    NSString *value = [NSString stringWithUTF8String:json];
    free(json);
    NSArray *items = [NSJSONSerialization JSONObjectWithData:[value dataUsingEncoding:NSUTF8StringEncoding] options:0 error:NULL];
    return [items isKindOfClass:NSArray.class] ? items : @[];
}

- (void)linkMenuQueryChanged {
    if (_suppress || _text.hasMarkedText) return;
    BOOL mention = NO;
    NSRange range = [self linkMenuQueryRange:&mention];
    if (range.location == NSNotFound || (_linkPanel && mention != _linkMention)) {
        [self linkMenuClose];
        if (range.location == NSNotFound) return;
    }
    // A card waits on the "@" the slash menu typed, nowhere else.
    if (_linkCard && (!mention || range.location != _linkCardAt)) _linkCard = NO;
    [self slashClose];
    [self emojiMenuClose];
    [self formatBarHide];
    _linkMention = mention;
    NSUInteger opener = mention ? 1 : 2;
    NSString *filter = [_text.string substringWithRange:NSMakeRange(range.location + opener, range.length - opener)];
    [self linkMenuOpenWithFilter:filter range:range];
}

- (void)linkMenuOpenWithFilter:(NSString *)filter range:(NSRange)range {
    if (![filter isEqualToString:_linkFilter]) {
        _linkFilter = [filter copy];
        _linkSelected = 0;
    }
    _linkQuery = range;
    NSArray *items = [self linkMenuItemsForFilter:_linkFilter];
    if (_linkMention && !_linkCard && !items.count) {
        // A typed "@handle" that names no note is just text.
        [self linkMenuClose];
        return;
    }
    if (!_linkPanel) {
        CRLinkMenuView *view = [[CRLinkMenuView alloc] initWithOwner:self];
        CRLinkPanel *panel = [[CRLinkPanel alloc] initWithContentRect:NSMakeRect(0, 0, CRLinkWidth, 120)
                                                           styleMask:NSWindowStyleMaskBorderless | NSWindowStyleMaskNonactivatingPanel
                                                             backing:NSBackingStoreBuffered defer:NO];
        panel.releasedWhenClosed = NO;
        panel.opaque = NO;
        panel.backgroundColor = NSColor.clearColor;
        panel.hasShadow = YES;
        panel.alphaValue = 0;
        panel.contentView = view;
        _linkPanel = panel;
        _linkView = view;
        _scroll.contentView.postsBoundsChangedNotifications = YES;
        _scroll.postsFrameChangedNotifications = YES;
        [[NSNotificationCenter defaultCenter] addObserver:self selector:@selector(linkMenuParentChanged:) name:NSViewBoundsDidChangeNotification object:_scroll.contentView];
        [[NSNotificationCenter defaultCenter] addObserver:self selector:@selector(linkMenuParentChanged:) name:NSViewFrameDidChangeNotification object:_scroll];
        [[NSNotificationCenter defaultCenter] addObserver:self selector:@selector(linkMenuClose) name:NSWindowDidResignKeyNotification object:_window];
        [[NSNotificationCenter defaultCenter] addObserver:self selector:@selector(linkMenuClose) name:NSApplicationDidResignActiveNotification object:nil];
        [_window addChildWindow:panel ordered:NSWindowAbove];
    }
    _linkPanel.appearance = [NSAppearance appearanceNamed:_dark ? NSAppearanceNameDarkAqua : NSAppearanceNameAqua];
    _linkView.items = items;
    if (_linkSelected >= (NSInteger)items.count) _linkSelected = 0;
    _linkView.selected = _linkSelected;
    _linkView.hovered = -1;
    [_linkPanel setContentSize:_linkView.bounds.size];
    [_linkPanel orderFront:nil];
    [self linkMenuPosition];
    if (_linkPanel.alphaValue < 1) {
        [NSAnimationContext runAnimationGroup:^(NSAnimationContext *context) {
            context.duration = 0.09;
            self->_linkPanel.animator.alphaValue = 1;
        }];
    }
}

- (void)linkMenuPosition {
    if (!_linkPanel || !_window) return;
    NSRange query = [self linkMenuQueryRange];
    if (query.location == NSNotFound) return;
    NSLayoutManager *layout = _text.layoutManager;
    [layout ensureLayoutForTextContainer:_text.textContainer];
    NSUInteger glyph = [layout glyphIndexForCharacterAtIndex:MIN(NSMaxRange(query), _text.string.length)];
    NSRect caret = [layout boundingRectForGlyphRange:NSMakeRange(glyph, 0) inTextContainer:_text.textContainer];
    caret = NSOffsetRect(caret, _text.textContainerOrigin.x, _text.textContainerOrigin.y);
    NSRect anchor = [_window convertRectToScreen:[_text convertRect:caret toView:nil]];
    NSRect windowFrame = _window.frame;
    CGFloat width = _linkView.bounds.size.width, height = _linkView.bounds.size.height;
    CGFloat x = MAX(NSMinX(windowFrame) + 4, MIN(anchor.origin.x, NSMaxX(windowFrame) - width - 4));
    CGFloat y = NSMinY(anchor) - height - 6;
    if (y < NSMinY(windowFrame) + 4) y = NSMaxY(anchor) + 6;
    y = MAX(NSMinY(windowFrame) + 4, MIN(y, NSMaxY(windowFrame) - height - 4));
    [_linkPanel setFrame:NSMakeRect(x, y, width, height) display:YES];
}

- (void)linkMenuParentChanged:(NSNotification *)note {
    if (!_linkPanel) return;
    NSRange query = [self linkMenuQueryRange];
    if (query.location == NSNotFound) { [self linkMenuClose]; return; }
    NSLayoutManager *layout = _text.layoutManager;
    [layout ensureLayoutForTextContainer:_text.textContainer];
    NSUInteger glyph = [layout glyphIndexForCharacterAtIndex:MIN(NSMaxRange(query), _text.string.length)];
    NSRect line = glyph < layout.numberOfGlyphs ? [layout lineFragmentRectForGlyphAtIndex:glyph effectiveRange:NULL] : layout.extraLineFragmentRect;
    line = NSOffsetRect(line, _text.textContainerOrigin.x, _text.textContainerOrigin.y);
    if (!NSIntersectsRect(line, _text.visibleRect)) { [self linkMenuClose]; return; }
    [self linkMenuPosition];
}

- (BOOL)linkMenuHandleKey:(NSEvent *)event {
    if (!_linkPanel || _text.hasMarkedText) return NO;
    NSEventModifierFlags flags = event.modifierFlags & NSEventModifierFlagDeviceIndependentFlagsMask;
    flags &= ~(NSEventModifierFlagCapsLock | NSEventModifierFlagNumericPad | NSEventModifierFlagFunction);
    if (flags) return NO;
    NSString *key = event.charactersIgnoringModifiers;
    if (key.length != 1) return NO;
    unichar c = [key characterAtIndex:0];
    if (c == 0xF700) { [self linkMenuMove:-1]; return YES; }
    if (c == 0xF701) { [self linkMenuMove:1]; return YES; }
    if (c == '\r' || c == 0x03) { [self linkMenuConfirm]; return YES; }
    if (c == 0x1B) { [self linkMenuClose]; return YES; }
    return NO;
}

- (void)linkMenuMove:(NSInteger)delta {
    NSInteger count = (NSInteger)_linkView.items.count;
    if (!count) return;
    _linkSelected = (NSInteger)(_linkSelected + delta + count) % count;
    _linkView.selected = _linkSelected;
}

- (void)linkMenuConfirm {
    NSArray *items = _linkView.items;
    if (_linkSelected < 0 || _linkSelected >= (NSInteger)items.count) return;
    [self linkMenuChoose:items[_linkSelected]];
}

- (void)linkMenuChooseIndex:(NSInteger)index {
    NSArray *items = _linkView.items;
    if (index < 0 || index >= (NSInteger)items.count) return;
    [self linkMenuChoose:items[index]];
}

- (void)linkMenuChoose:(NSDictionary *)item {
    if (!item || !_linkPanel) return;
    NSRange range = _linkQuery;
    BOOL mention = _linkMention, card = _linkCard && range.location == _linkCardAt;
    [self linkMenuClose];
    if (!_text.editable || _text.hasMarkedText || range.location == NSNotFound) return;
    NSString *title = item[@"title"] ?: @"";
    NSString *path = item[@"path"] ?: @"";
    if (!path.length) return;
    if (mention) {
        title = [title stringByReplacingOccurrencesOfString:@"\n" withString:@" "];
        if (!title.length) title = path.lastPathComponent.stringByDeletingPathExtension;
        if (card && [self insertPageCard:title path:path replacing:range]) return;
        [self insertMention:title path:path replacing:range];
        return;
    }
    // The link is inserted as a rendered run, not as literal Markdown: the
    // serializer then writes [title](path) itself and the text stays clickable.
    title = [title stringByReplacingOccurrencesOfString:@"\n" withString:@" "];
    if (!title.length) title = path;
    // The bracket auto-pair leaves "]]" after the query; take them with it so
    // the inserted link is not followed by stray brackets.
    NSUInteger end = NSMaxRange(range);
    NSUInteger consumed = 0;
    while (end < _text.string.length && consumed < 2 && [_text.string characterAtIndex:end] == ']') { end++; consumed++; }
    if (consumed) range = NSMakeRange(range.location, end - range.location);
    NSDictionary *attrs = range.location < _text.string.length
        ? [_text.textStorage attributesAtIndex:range.location effectiveRange:NULL]
        : _text.typingAttributes;
    NSMutableDictionary *plain = [attrs mutableCopy];
    for (NSString *key in @[CRBold, CRItalic, CRCode, CRUnderline, CRKbd, CRHighlight, CRSup, CRSub,
                            NSStrikethroughStyleAttributeName, NSLinkAttributeName, CRLinkTitle]) {
        [plain removeObjectForKey:key];
    }
    plain[NSLinkAttributeName] = path;
    [self beginStructuralCommand];
    [_undo beginUndoGrouping];
    [self replace:range with:[[NSAttributedString alloc] initWithString:title attributes:plain]
       selection:NSMakeRange(range.location + title.length, 0)];
    _text.typingAttributes = [[self visual:@{CRParagraph: CRParagraphAt(self, range.location)} paragraph:CRParagraphAt(self, range.location)] mutableCopy];
    [_undo endUndoGrouping];
}

- (void)linkMenuClose {
    _linkCard = NO;
    if (!_linkPanel) return;
    [[NSNotificationCenter defaultCenter] removeObserver:self name:NSViewBoundsDidChangeNotification object:_scroll.contentView];
    [[NSNotificationCenter defaultCenter] removeObserver:self name:NSViewFrameDidChangeNotification object:_scroll];
    [[NSNotificationCenter defaultCenter] removeObserver:self name:NSWindowDidResignKeyNotification object:_window];
    [[NSNotificationCenter defaultCenter] removeObserver:self name:NSApplicationDidResignActiveNotification object:nil];
    [_window removeChildWindow:_linkPanel];
    [_linkPanel close];
    _linkPanel = nil;
    _linkView = nil;
    _linkFilter = nil;
    _linkSelected = 0;
    _linkQuery = NSMakeRange(NSNotFound, 0);
}

- (BOOL)linkMenuSnapshotTo:(NSString *)path {
    if (!_linkView) return NO;
    NSRect bounds = _linkView.bounds;
    NSBitmapImageRep *rep = [_linkView bitmapImageRepForCachingDisplayInRect:bounds];
    if (!rep) return NO;
    [_linkView cacheDisplayInRect:bounds toBitmapImageRep:rep];
    NSData *png = [rep representationUsingType:NSBitmapImageFileTypePNG properties:@{}];
    return png && [png writeToFile:path atomically:YES];
}

- (NSDictionary *)linkMenuState {
    NSMutableArray *titles = [NSMutableArray array];
    for (NSDictionary *item in _linkView.items ?: @[]) [titles addObject:item[@"title"] ?: @""];
    return @{@"open": (_linkPanel ? @YES : @NO), @"filter": _linkFilter ?: @"", @"selected": @(_linkSelected), @"items": titles,
             @"mention": (_linkPanel && _linkMention ? @YES : @NO), @"card": (_linkPanel && _linkCard ? @YES : @NO)};
}

// MARK: - Page mentions

// insertMention replaces the typed "@query" with a "@Title" chip linked to the
// note, followed by a plain space so typing carries on outside it.
- (void)insertMention:(NSString *)title path:(NSString *)path replacing:(NSRange)range {
    NSDictionary *attrs = range.location < _text.string.length
        ? [_text.textStorage attributesAtIndex:range.location effectiveRange:NULL]
        : _text.typingAttributes;
    NSMutableDictionary *plain = [attrs mutableCopy];
    for (NSString *key in @[CRBold, CRItalic, CRCode, CRUnderline, CRKbd, CRHighlight, CRSup, CRSub, CRColor, CRBackground,
                            NSStrikethroughStyleAttributeName, NSLinkAttributeName, CRLinkTitle, CRMention, CRInlineSource]) {
        [plain removeObjectForKey:key];
    }
    NSDictionary *paragraph = CRParagraphAt(self, range.location);
    NSMutableDictionary *chip = [plain mutableCopy];
    chip[NSLinkAttributeName] = path;
    chip[CRMention] = @YES;
    NSMutableAttributedString *value = [[NSMutableAttributedString alloc] initWithString:[@"@" stringByAppendingString:title] attributes:[self visual:chip paragraph:paragraph]];
    // An existing space after the query keeps its place; otherwise add one.
    NSUInteger end = NSMaxRange(range);
    BOOL spaced = end < _text.string.length && [[NSCharacterSet whitespaceAndNewlineCharacterSet] characterIsMember:[_text.string characterAtIndex:end]];
    if (!spaced) [value appendAttributedString:[[NSAttributedString alloc] initWithString:@" " attributes:[self visual:plain paragraph:paragraph]]];
    [self beginStructuralCommand];
    [_undo beginUndoGrouping];
    [self replace:range with:value selection:NSMakeRange(range.location + value.length + (spaced ? 1 : 0), 0)];
    _text.typingAttributes = [[self visual:plain paragraph:CRParagraphAt(self, range.location)] mutableCopy];
    [_undo endUndoGrouping];
}

// mentionRangeAt is the whole mention covering location, or NSNotFound.
- (NSRange)mentionRangeAt:(NSUInteger)location {
    if (location >= _text.string.length || ![[_text.textStorage attribute:CRMention atIndex:location effectiveRange:NULL] boolValue]) return NSMakeRange(NSNotFound, 0);
    id link = [_text.textStorage attribute:NSLinkAttributeName atIndex:location effectiveRange:NULL];
    if (!link) return NSMakeRange(NSNotFound, 0);
    NSRange mention, linked;
    [_text.textStorage attribute:CRMention atIndex:location longestEffectiveRange:&mention inRange:CRContent(_text.string, [_text.string paragraphRangeForRange:NSMakeRange(location, 0)])];
    [_text.textStorage attribute:NSLinkAttributeName atIndex:location longestEffectiveRange:&linked inRange:mention];
    return linked;
}

// deleteMentionBefore removes a whole mention when Backspace lands on its
// last character: a page reference goes away in one piece, as in Notion.
- (BOOL)deleteMentionBefore:(NSUInteger)location {
    NSRange mention = [self mentionRangeAt:location - 1];
    if (mention.location == NSNotFound || NSMaxRange(mention) != location) return NO;
    [self beginStructuralCommand];
    [self replace:mention with:[NSAttributedString new] selection:NSMakeRange(mention.location, 0)];
    return YES;
}

// openDocumentLink follows a mention or any note link inside the app.
- (BOOL)openDocumentLink:(id)link {
    if (!_callback || !CRLocalDocument(link)) return NO;
    NSString *target = [link isKindOfClass:NSURL.class] ? [(NSURL *)link absoluteString] : [link description];
    return cr_open_document(_callback, target.UTF8String) != 0;
}

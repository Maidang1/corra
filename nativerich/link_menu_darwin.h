// The link menu's editor half: typing "[[" opens note suggestions beside the
// caret so a note can be linked without typing its path. Included inside
// @implementation CREditor.

#pragma once

// linkMenuQueryRange locates a "[[query" the caret sits at the end of. The
// query may contain spaces (note titles do); it stops at brackets or a line
// break. Returns {NSNotFound, 0} elsewhere and inside protected blocks.
- (NSRange)linkMenuQueryRange {
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
        CRKind(info, @"toc") || CRCellAt(self, content.location) || [info[@"footnoteId"] length]) {
        return NSMakeRange(NSNotFound, 0);
    }
    NSString *prefix = [string substringWithRange:NSMakeRange(content.location, caret - content.location)];
    NSRange opener = [prefix rangeOfString:@"[[" options:NSBackwardsSearch];
    if (opener.location == NSNotFound) return NSMakeRange(NSNotFound, 0);
    NSString *query = [prefix substringFromIndex:NSMaxRange(opener)];
    if (query.length > 40) return NSMakeRange(NSNotFound, 0);
    if ([query rangeOfCharacterFromSet:[NSCharacterSet characterSetWithCharactersInString:@"[]\n\r"]].location != NSNotFound) {
        return NSMakeRange(NSNotFound, 0);
    }
    NSUInteger start = content.location + opener.location;
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
    NSRange range = [self linkMenuQueryRange];
    if (range.location == NSNotFound) {
        [self linkMenuClose];
        return;
    }
    [self slashClose];
    [self emojiMenuClose];
    [self formatBarHide];
    NSString *filter = [_text.string substringWithRange:NSMakeRange(range.location + 2, range.length - 2)];
    [self linkMenuOpenWithFilter:filter range:range];
}

- (void)linkMenuOpenWithFilter:(NSString *)filter range:(NSRange)range {
    if (![filter isEqualToString:_linkFilter]) {
        _linkFilter = [filter copy];
        _linkSelected = 0;
    }
    _linkQuery = range;
    NSArray *items = [self linkMenuItemsForFilter:_linkFilter];
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
    [self linkMenuClose];
    if (!_text.editable || _text.hasMarkedText || range.location == NSNotFound) return;
    NSString *title = item[@"title"] ?: @"";
    NSString *path = item[@"path"] ?: @"";
    if (!path.length) return;
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
    return @{@"open": (_linkPanel ? @YES : @NO), @"filter": _linkFilter ?: @"", @"selected": @(_linkSelected), @"items": titles};
}

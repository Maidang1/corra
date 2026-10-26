// The emoji menu's editor half: typing ":" at a word start opens a shortcode
// list beside the caret; typing filters, arrows navigate, Return inserts and
// Escape dismisses. Included inside @implementation CREditor.

#pragma once

// emojiMenuQueryRange locates a ":query" the caret sits at the end of, where
// the colon starts a word. Returns {NSNotFound, 0} elsewhere.
- (NSRange)emojiMenuQueryRange {
    if (!_text.editable || _text.hasMarkedText || _suppress) return NSMakeRange(NSNotFound, 0);
    if (_text.selectedRange.length) return NSMakeRange(NSNotFound, 0);
    NSString *string = _text.string;
    NSUInteger caret = MIN(_text.selectedRange.location, string.length);
    NSRange paragraph = [string paragraphRangeForRange:NSMakeRange(caret, 0)];
    NSRange content = CRContent(string, paragraph);
    if (!content.length || caret < content.location || caret > NSMaxRange(content)) return NSMakeRange(NSNotFound, 0);
    NSDictionary *paragraphInfo = CRParagraphAt(self, content.location);
    if (CRKind(paragraphInfo, @"code") || CRKind(paragraphInfo, @"source") || CRKind(paragraphInfo, @"opaque") ||
        CRKind(paragraphInfo, @"math") || CRKind(paragraphInfo, @"image") || CRKind(paragraphInfo, @"diagram") ||
        CRKind(paragraphInfo, @"toc") || CRKind(paragraphInfo, @"page") || CRCellAt(self, content.location) || [paragraphInfo[@"footnoteId"] length]) {
        return NSMakeRange(NSNotFound, 0);
    }
    NSUInteger index = caret;
    while (index > content.location) {
        unichar c = [string characterAtIndex:index - 1];
        if (c == ':') {
            for (NSUInteger i = index; i < caret; i++) {
                unichar q = [string characterAtIndex:i];
                if (!((q >= 'a' && q <= 'z') || (q >= '0' && q <= '9') || q == '_' || q == '+' || q == '-')) {
                    return NSMakeRange(NSNotFound, 0);
                }
            }
            if (index - 1 > content.location) {
                unichar before = [string characterAtIndex:index - 2];
                BOOL boundary = [[NSCharacterSet whitespaceAndNewlineCharacterSet] characterIsMember:before] ||
                                before == '(' || before == 0xFF08 || before == '[' || before == 0x3010 ||
                                before == '"' || before == 0x201C;
                if (!boundary) return NSMakeRange(NSNotFound, 0);
            }
            if (caret - (index - 1) > 26) return NSMakeRange(NSNotFound, 0);
            return NSMakeRange(index - 1, caret - (index - 1));
        }
        if (!((c >= 'a' && c <= 'z') || (c >= '0' && c <= '9') || c == '_' || c == '+' || c == '-')) {
            return NSMakeRange(NSNotFound, 0);
        }
        index--;
    }
    return NSMakeRange(NSNotFound, 0);
}

- (NSString *)emojiMenuFilterText {
    return _emojiFilter ?: @"";
}

- (NSArray *)emojiMenuItemsForFilter:(NSString *)filter {
    char *json = cr_emoji_search((char *)(filter ?: @"").UTF8String);
    if (!json) return @[];
    NSString *value = [NSString stringWithUTF8String:json];
    free(json);
    NSArray *items = [NSJSONSerialization JSONObjectWithData:[value dataUsingEncoding:NSUTF8StringEncoding] options:0 error:NULL];
    return [items isKindOfClass:NSArray.class] ? items : @[];
}

- (void)emojiMenuQueryChanged {
    if (_suppress || _text.hasMarkedText) return;
    NSRange range = [self emojiMenuQueryRange];
    if (range.location == NSNotFound) {
        [self emojiMenuClose];
        return;
    }
    [self slashClose];
    [self linkMenuClose];
    [self formatBarHide];
    NSString *filter = [_text.string substringWithRange:NSMakeRange(range.location + 1, range.length - 1)];
    [self emojiMenuOpenWithFilter:filter range:range];
}

- (void)emojiMenuOpenWithFilter:(NSString *)filter range:(NSRange)range {
    if (![filter isEqualToString:_emojiFilter]) {
        _emojiFilter = [filter copy];
        _emojiSelected = 0;
    }
    _emojiQuery = range;
    NSArray *items = [self emojiMenuItemsForFilter:_emojiFilter];
    if (!_emojiPanel) {
        CREmojiMenuView *view = [[CREmojiMenuView alloc] initWithOwner:self];
        CREmojiPanel *panel = [[CREmojiPanel alloc] initWithContentRect:NSMakeRect(0, 0, CREmojiWidth, 120)
                                                             styleMask:NSWindowStyleMaskBorderless | NSWindowStyleMaskNonactivatingPanel
                                                               backing:NSBackingStoreBuffered defer:NO];
        panel.releasedWhenClosed = NO;
        panel.opaque = NO;
        panel.backgroundColor = NSColor.clearColor;
        panel.hasShadow = YES;
        panel.alphaValue = 0;
        panel.contentView = view;
        _emojiPanel = panel;
        _emojiView = view;
        _scroll.contentView.postsBoundsChangedNotifications = YES;
        _scroll.postsFrameChangedNotifications = YES;
        [[NSNotificationCenter defaultCenter] addObserver:self selector:@selector(emojiMenuParentChanged:) name:NSViewBoundsDidChangeNotification object:_scroll.contentView];
        [[NSNotificationCenter defaultCenter] addObserver:self selector:@selector(emojiMenuParentChanged:) name:NSViewFrameDidChangeNotification object:_scroll];
        [[NSNotificationCenter defaultCenter] addObserver:self selector:@selector(emojiMenuClose) name:NSWindowDidResignKeyNotification object:_window];
        [[NSNotificationCenter defaultCenter] addObserver:self selector:@selector(emojiMenuClose) name:NSApplicationDidResignActiveNotification object:nil];
        [_window addChildWindow:panel ordered:NSWindowAbove];
    }
    _emojiPanel.appearance = [NSAppearance appearanceNamed:_dark ? NSAppearanceNameDarkAqua : NSAppearanceNameAqua];
    _emojiView.items = items;
    if (_emojiSelected >= (NSInteger)items.count) _emojiSelected = 0;
    _emojiView.selected = _emojiSelected;
    _emojiView.hovered = -1;
    [_emojiPanel setContentSize:_emojiView.bounds.size];
    [_emojiPanel orderFront:nil];
    [self emojiMenuPosition];
    if (_emojiPanel.alphaValue < 1) {
        [NSAnimationContext runAnimationGroup:^(NSAnimationContext *context) {
            context.duration = 0.09;
            self->_emojiPanel.animator.alphaValue = 1;
        }];
    }
}

- (void)emojiMenuPosition {
    if (!_emojiPanel || !_window) return;
    NSRange query = [self emojiMenuQueryRange];
    if (query.location == NSNotFound) return;
    NSLayoutManager *layout = _text.layoutManager;
    [layout ensureLayoutForTextContainer:_text.textContainer];
    NSUInteger glyph = [layout glyphIndexForCharacterAtIndex:MIN(NSMaxRange(query), _text.string.length)];
    NSRect caret = [layout boundingRectForGlyphRange:NSMakeRange(glyph, 0) inTextContainer:_text.textContainer];
    caret = NSOffsetRect(caret, _text.textContainerOrigin.x, _text.textContainerOrigin.y);
    NSRect anchor = [_window convertRectToScreen:[_text convertRect:caret toView:nil]];
    NSRect windowFrame = _window.frame;
    CGFloat width = _emojiView.bounds.size.width, height = _emojiView.bounds.size.height;
    CGFloat x = MAX(NSMinX(windowFrame) + 4, MIN(anchor.origin.x, NSMaxX(windowFrame) - width - 4));
    CGFloat y = NSMinY(anchor) - height - 6;
    if (y < NSMinY(windowFrame) + 4) y = NSMaxY(anchor) + 6;
    y = MAX(NSMinY(windowFrame) + 4, MIN(y, NSMaxY(windowFrame) - height - 4));
    [_emojiPanel setFrame:NSMakeRect(x, y, width, height) display:YES];
}

- (void)emojiMenuParentChanged:(NSNotification *)note {
    if (!_emojiPanel) return;
    NSRange query = [self emojiMenuQueryRange];
    if (query.location == NSNotFound) { [self emojiMenuClose]; return; }
    NSLayoutManager *layout = _text.layoutManager;
    [layout ensureLayoutForTextContainer:_text.textContainer];
    NSUInteger glyph = [layout glyphIndexForCharacterAtIndex:MIN(NSMaxRange(query), _text.string.length)];
    NSRect line = glyph < layout.numberOfGlyphs ? [layout lineFragmentRectForGlyphAtIndex:glyph effectiveRange:NULL] : layout.extraLineFragmentRect;
    line = NSOffsetRect(line, _text.textContainerOrigin.x, _text.textContainerOrigin.y);
    if (!NSIntersectsRect(line, _text.visibleRect)) { [self emojiMenuClose]; return; }
    [self emojiMenuPosition];
}

- (BOOL)emojiMenuHandleKey:(NSEvent *)event {
    if (!_emojiPanel || _text.hasMarkedText) return NO;
    NSEventModifierFlags flags = event.modifierFlags & NSEventModifierFlagDeviceIndependentFlagsMask;
    flags &= ~(NSEventModifierFlagCapsLock | NSEventModifierFlagNumericPad | NSEventModifierFlagFunction);
    if (flags) return NO;
    NSString *key = event.charactersIgnoringModifiers;
    if (key.length != 1) return NO;
    unichar c = [key characterAtIndex:0];
    if (c == 0xF700) { [self emojiMenuMove:-1]; return YES; }
    if (c == 0xF701) { [self emojiMenuMove:1]; return YES; }
    if (c == '\r' || c == 0x03) { [self emojiMenuConfirm]; return YES; }
    if (c == 0x1B) { [self emojiMenuClose]; return YES; }
    return NO;
}

- (void)emojiMenuMove:(NSInteger)delta {
    NSInteger count = (NSInteger)_emojiView.items.count;
    if (!count) return;
    _emojiSelected = (NSInteger)(_emojiSelected + delta + count) % count;
    _emojiView.selected = _emojiSelected;
}

- (void)emojiMenuConfirm {
    NSArray *items = _emojiView.items;
    if (_emojiSelected < 0 || _emojiSelected >= (NSInteger)items.count) return;
    [self emojiMenuChoose:items[_emojiSelected]];
}

- (void)emojiMenuChooseIndex:(NSInteger)index {
    NSArray *items = _emojiView.items;
    if (index < 0 || index >= (NSInteger)items.count) return;
    [self emojiMenuChoose:items[index]];
}

- (void)emojiMenuChoose:(NSDictionary *)item {
    if (!item || !_emojiPanel) return;
    NSRange range = _emojiQuery;
    [self emojiMenuClose];
    if (!_text.editable || _text.hasMarkedText) return;
    NSString *emoji = item[@"emoji"];
    if (!emoji.length || range.location == NSNotFound) return;
    NSDictionary *attrs = range.location < _text.string.length
        ? [_text.textStorage attributesAtIndex:range.location effectiveRange:NULL]
        : _text.typingAttributes;
    [self beginStructuralCommand];
    [_undo beginUndoGrouping];
    [self replace:range with:[[NSAttributedString alloc] initWithString:emoji attributes:attrs]
       selection:NSMakeRange(range.location + emoji.length, 0)];
    [_undo endUndoGrouping];
}

- (void)emojiMenuClose {
    if (!_emojiPanel) return;
    [[NSNotificationCenter defaultCenter] removeObserver:self name:NSViewBoundsDidChangeNotification object:_scroll.contentView];
    [[NSNotificationCenter defaultCenter] removeObserver:self name:NSViewFrameDidChangeNotification object:_scroll];
    [[NSNotificationCenter defaultCenter] removeObserver:self name:NSWindowDidResignKeyNotification object:_window];
    [[NSNotificationCenter defaultCenter] removeObserver:self name:NSApplicationDidResignActiveNotification object:nil];
    [_window removeChildWindow:_emojiPanel];
    [_emojiPanel close];
    _emojiPanel = nil;
    _emojiView = nil;
    _emojiFilter = nil;
    _emojiSelected = 0;
    _emojiQuery = NSMakeRange(NSNotFound, 0);
}

// emojiMenuSnapshotTo renders the palette to a PNG for visual checks.
- (BOOL)emojiMenuSnapshotTo:(NSString *)path {
    if (!_emojiView) return NO;
    NSRect bounds = _emojiView.bounds;
    NSBitmapImageRep *rep = [_emojiView bitmapImageRepForCachingDisplayInRect:bounds];
    if (!rep) return NO;
    [_emojiView cacheDisplayInRect:bounds toBitmapImageRep:rep];
    NSData *png = [rep representationUsingType:NSBitmapImageFileTypePNG properties:@{}];
    return png && [png writeToFile:path atomically:YES];
}

- (NSDictionary *)emojiMenuState {
    NSMutableArray *names = [NSMutableArray array];
    for (NSDictionary *item in _emojiView.items ?: @[]) [names addObject:item[@"name"] ?: @""];
    return @{@"open": (_emojiPanel ? @YES : @NO), @"filter": _emojiFilter ?: @"", @"selected": @(_emojiSelected), @"items": names};
}

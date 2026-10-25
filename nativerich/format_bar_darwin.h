// The floating format bar's editor half: shows a pill over a prose selection,
// applies inline traits through the same format: transactions as the menu,
// and prompts for a link with a native sheet. Included inside
// @implementation CREditor (after slash_menu_darwin.h).

#pragma once

// MARK: - Format bar

// CRFormatBarBlockedKind: blocks whose text is not inline prose, so the bar
// would only offer formats the serializer cannot express.
static BOOL CRFormatBarBlockedKind(NSDictionary *p) {
    return CRKind(p, @"code") || CRKind(p, @"source") || CRKind(p, @"opaque") ||
           CRKind(p, @"math") || CRKind(p, @"image") || CRKind(p, @"diagram") || CRKind(p, @"toc");
}

- (NSRange)formatBarSelection {
    if (!_text.editable || _text.hasMarkedText || _suppress) return NSMakeRange(NSNotFound, 0);
    NSRange selection = _text.selectedRange;
    if (!selection.length || NSMaxRange(selection) > _text.string.length) return NSMakeRange(NSNotFound, 0);
    if ([_text.textStorage attribute:NSAttachmentAttributeName atIndex:selection.location effectiveRange:NULL]) return NSMakeRange(NSNotFound, 0);
    for (NSValue *value in CRSelectedParagraphs(self)) {
        if (CRFormatBarBlockedKind(CRParagraphAt(self, value.rangeValue.location))) return NSMakeRange(NSNotFound, 0);
    }
    return selection;
}

// formatBarActiveCommands lists the traits every run of the selection carries,
// so the pill shows exactly what is on.
- (NSSet *)formatBarActiveCommands {
    NSRange selection = [self formatBarSelection];
    if (selection.location == NSNotFound) return [NSSet set];
    NSArray *commands = @[@"bold", @"italic", @"underline", @"strike", @"code", @"highlight", @"link"];
    NSDictionary *keys = @{@"bold": CRBold, @"italic": CRItalic, @"underline": CRUnderline,
                           @"strike": NSStrikethroughStyleAttributeName, @"code": CRCode,
                           @"highlight": CRHighlight, @"link": NSLinkAttributeName};
    NSMutableSet *active = [NSMutableSet set];
    for (NSString *command in commands) {
        NSString *key = keys[command];
        __block BOOL all = YES;
        [_text.textStorage enumerateAttributesInRange:selection options:0 usingBlock:^(NSDictionary *attrs, NSRange range, BOOL *stop) {
            BOOL on = [key isEqual:NSLinkAttributeName] ? attrs[NSLinkAttributeName] != nil : [attrs[key] boolValue];
            if (!on) { all = NO; *stop = YES; }
        }];
        if (all) [active addObject:command];
    }
    return active;
}

- (void)updateFormatBar {
    NSRange selection = [self formatBarSelection];
    if (selection.location == NSNotFound) {
        [self formatBarHide];
        return;
    }
    if (!_formatPanel) {
        CRFormatBarView *view = [[CRFormatBarView alloc] initWithOwner:self];
        CRFormatBarPanel *panel = [[CRFormatBarPanel alloc] initWithContentRect:NSMakeRect(0, 0, view.bounds.size.width, view.bounds.size.height)
                                                                     styleMask:NSWindowStyleMaskBorderless | NSWindowStyleMaskNonactivatingPanel
                                                                       backing:NSBackingStoreBuffered defer:NO];
        panel.releasedWhenClosed = NO;
        panel.opaque = NO;
        panel.backgroundColor = NSColor.clearColor;
        panel.hasShadow = YES;
        panel.alphaValue = 0;
        panel.contentView = view;
        _formatPanel = panel;
        _formatView = view;
        _scroll.contentView.postsBoundsChangedNotifications = YES;
        _scroll.postsFrameChangedNotifications = YES;
        [[NSNotificationCenter defaultCenter] addObserver:self selector:@selector(formatBarParentChanged:) name:NSViewBoundsDidChangeNotification object:_scroll.contentView];
        [[NSNotificationCenter defaultCenter] addObserver:self selector:@selector(formatBarParentChanged:) name:NSViewFrameDidChangeNotification object:_scroll];
        [[NSNotificationCenter defaultCenter] addObserver:self selector:@selector(formatBarHide) name:NSWindowDidResignKeyNotification object:_window];
        [[NSNotificationCenter defaultCenter] addObserver:self selector:@selector(formatBarHide) name:NSApplicationDidResignActiveNotification object:nil];
        [_window addChildWindow:panel ordered:NSWindowAbove];
    }
    _formatPanel.appearance = [NSAppearance appearanceNamed:_dark ? NSAppearanceNameDarkAqua : NSAppearanceNameAqua];
    _formatView.active = [self formatBarActiveCommands];
    [_formatPanel setContentSize:_formatView.bounds.size];
    [_formatPanel orderFront:nil];
    [self formatBarPositionWithSelection:selection];
    // A short fade keeps the pill from popping over a fresh selection.
    if (_formatPanel.alphaValue < 1) {
        [NSAnimationContext runAnimationGroup:^(NSAnimationContext *context) {
            context.duration = 0.09;
            _formatPanel.animator.alphaValue = 1;
        }];
    }
}

- (void)formatBarHide {
    if (!_formatPanel) return;
    [[NSNotificationCenter defaultCenter] removeObserver:self name:NSViewBoundsDidChangeNotification object:_scroll.contentView];
    [[NSNotificationCenter defaultCenter] removeObserver:self name:NSViewFrameDidChangeNotification object:_scroll];
    [[NSNotificationCenter defaultCenter] removeObserver:self name:NSWindowDidResignKeyNotification object:_window];
    [[NSNotificationCenter defaultCenter] removeObserver:self name:NSApplicationDidResignActiveNotification object:nil];
    [_window removeChildWindow:_formatPanel];
    [_formatPanel close];
    _formatPanel = nil;
    _formatView = nil;
}

- (void)formatBarParentChanged:(NSNotification *)note {
    if (!_formatPanel) return;
    NSRange selection = [self formatBarSelection];
    if (selection.location == NSNotFound) { [self formatBarHide]; return; }
    [self formatBarPositionWithSelection:selection];
}

// formatBarAnchorScreen is the selection's first line, in screen coordinates;
// NSZeroRect when the line is not laid out or has scrolled out of sight.
- (NSRect)formatBarAnchorScreen:(NSRange)selection {
    if (selection.location == NSNotFound || !_window) return NSZeroRect;
    NSLayoutManager *layout = _text.layoutManager;
    [layout ensureLayoutForTextContainer:_text.textContainer];
    NSRange glyph = [layout glyphRangeForCharacterRange:NSMakeRange(selection.location, 1) actualCharacterRange:NULL];
    if (!glyph.length && glyph.location >= layout.numberOfGlyphs) return NSZeroRect;
    NSRect line = [layout lineFragmentRectForGlyphAtIndex:glyph.location effectiveRange:NULL];
    line = NSOffsetRect(line, _text.textContainerOrigin.x, _text.textContainerOrigin.y);
    if (!NSIntersectsRect(line, _text.visibleRect)) return NSZeroRect;
    return [_window convertRectToScreen:[_text convertRect:line toView:nil]];
}

// formatBarPositionWithSelection centers the pill over the selection's first
// line, above it, flipping below when the line is too close to the top.
- (void)formatBarPositionWithSelection:(NSRange)selection {
    if (!_formatPanel || !_window) return;
    NSLayoutManager *layout = _text.layoutManager;
    [layout ensureLayoutForTextContainer:_text.textContainer];
    NSRange glyph = [layout glyphRangeForCharacterRange:NSMakeRange(selection.location, 1) actualCharacterRange:NULL];
    if (!glyph.length && glyph.location >= layout.numberOfGlyphs) { [self formatBarHide]; return; }
    NSRect line = [layout lineFragmentRectForGlyphAtIndex:glyph.location effectiveRange:NULL];
    NSRect used = [layout lineFragmentUsedRectForGlyphAtIndex:glyph.location effectiveRange:NULL];
    line = NSOffsetRect(line, _text.textContainerOrigin.x, _text.textContainerOrigin.y);
    used = NSOffsetRect(used, _text.textContainerOrigin.x, _text.textContainerOrigin.y);
    if (!NSIntersectsRect(line, _text.visibleRect)) { [self formatBarHide]; return; }
    NSRect lineScreen = [_window convertRectToScreen:[_text convertRect:line toView:nil]];
    NSRect usedScreen = [_window convertRectToScreen:[_text convertRect:used toView:nil]];
    NSRect windowFrame = _window.frame;
    CGFloat width = _formatView.bounds.size.width, height = _formatView.bounds.size.height;
    CGFloat x = NSMidX(usedScreen) - width / 2;
    x = MAX(NSMinX(windowFrame) + 6, MIN(x, NSMaxX(windowFrame) - width - 6));
    CGFloat y = NSMaxY(lineScreen) + 6;
    if (y + height > NSMaxY(windowFrame) - 6) y = NSMinY(lineScreen) - height - 6;
    y = MAX(NSMinY(windowFrame) + 6, MIN(y, NSMaxY(windowFrame) - height - 6));
    [_formatPanel setFrame:NSMakeRect(x, y, width, height) display:YES];
}

// formatBarApply routes a button (mouse) or a test hook through the same
// format transactions as the Format menu.
- (void)formatBarApply:(NSString *)command {
    if (![command isKindOfClass:NSString.class] || !command.length) return;
    if ([command isEqual:@"link"]) { [self formatBarPromptLink]; return; }
    if (!_text.editable || _text.hasMarkedText) return;
    [self format:command];
    [self updateFormatBar];
}

// formatBarLink wraps (or, for an empty URL, unwraps) the selection.
- (void)formatBarLink:(NSString *)url {
    [self formatBarLink:url range:_text.selectedRange];
}

- (void)formatBarLink:(NSString *)url range:(NSRange)selection {
    if (!_text.editable || _text.hasMarkedText || !selection.length) return;
    if (NSMaxRange(selection) > _text.string.length) return;
    NSMutableAttributedString *after = [_text.textStorage mutableCopy];
    if (url.length) [after addAttribute:NSLinkAttributeName value:url range:selection];
    else [after removeAttribute:NSLinkAttributeName range:selection];
    [self replace:NSMakeRange(0, _text.string.length) with:after selection:selection];
    [self updateFormatBar];
}

- (void)formatBarPromptLink {
    NSRange selection = [self formatBarSelection];
    if (selection.location == NSNotFound || !_window) return;
    id current = [_text.textStorage attribute:NSLinkAttributeName atIndex:selection.location effectiveRange:NULL];
    NSString *existing = [current isKindOfClass:NSString.class] ? current : nil;
    NSAlert *alert = [NSAlert new];
    alert.messageText = @"链接";
    alert.informativeText = @"输入网址并按“应用”；留空可移除链接。";
    [alert addButtonWithTitle:@"应用"];
    [alert addButtonWithTitle:@"取消"];
    NSTextField *field = [[NSTextField alloc] initWithFrame:NSMakeRect(0, 0, 300, 24)];
    field.stringValue = existing;
    field.placeholderString = @"https://";
    field.accessibilityLabel = @"链接地址";
    alert.accessoryView = field;
    alert.window.initialFirstResponder = field;
    NSUInteger revision = _documentRevision;
    NSString *snapshot = [_text.string copy];
    [alert beginSheetModalForWindow:_window completionHandler:^(NSModalResponse response) {
        if (response != NSAlertFirstButtonReturn) { [self.window makeFirstResponder:self.text]; return; }
        if (!self.window || self.documentRevision != revision || ![self.text.string isEqual:snapshot]) return;
        NSString *url = [field.stringValue stringByTrimmingCharactersInSet:NSCharacterSet.whitespaceAndNewlineCharacterSet];
        [self formatBarLink:url range:selection];
        [self.window makeFirstResponder:self.text];
    }];
}

// formatBarSnapshotTo renders the pill to a PNG for visual checks.
- (BOOL)formatBarSnapshotTo:(NSString *)path {
    if (!_formatView) return NO;
    NSRect bounds = _formatView.bounds;
    NSBitmapImageRep *rep = [_formatView bitmapImageRepForCachingDisplayInRect:bounds];
    if (!rep) return NO;
    [_formatView cacheDisplayInRect:bounds toBitmapImageRep:rep];
    NSData *png = [rep representationUsingType:NSBitmapImageFileTypePNG properties:@{}];
    return png && [png writeToFile:path atomically:YES];
}

// formatBarState exposes the bar to the Go test harness.
- (NSDictionary *)formatBarState {
    NSArray *active = [[[self formatBarActiveCommands] allObjects] sortedArrayUsingSelector:@selector(compare:)];
    NSMutableDictionary *state = [@{@"open": (_formatPanel ? @YES : @NO), @"active": active} mutableCopy];
    if (_formatPanel && _window) {
        state[@"frame"] = @{@"x": @(_formatPanel.frame.origin.x), @"y": @(_formatPanel.frame.origin.y),
                            @"w": @(_formatPanel.frame.size.width), @"h": @(_formatPanel.frame.size.height)};
        state[@"window"] = @{@"x": @(_window.frame.origin.x), @"y": @(_window.frame.origin.y),
                             @"w": @(_window.frame.size.width), @"h": @(_window.frame.size.height)};
        NSRect anchor = [self formatBarAnchorScreen:[self formatBarSelection]];
        state[@"anchor"] = @{@"x": @(anchor.origin.x), @"y": @(anchor.origin.y),
                             @"w": @(anchor.size.width), @"h": @(anchor.size.height)};
    }
    return state;
}

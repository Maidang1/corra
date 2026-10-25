// Slash menu: a Notion-style block picker. Typing "/" on an empty body line
// opens a panel under the caret; further characters filter the list, arrows
// navigate, Return applies the selected block, Escape dismisses. Included
// inside @implementation CREditor (after block_rules_darwin.h), so it can
// reuse format:/insertBlock:/setQuoteAlert: directly.

#pragma once

// MARK: - CREditor

// The methods below are included inside @implementation CREditor, so they
// share its ivars and transactions with the rest of the editor.

// slashBuildItems is the fixed block palette: title, Markdown spelling,
// SF Symbol fallback chain, lowercase filter keywords, editor command.
- (NSArray *)slashBuildItems {
    NSArray *raw = @[
        @[@"一级标题", @"#", @[], @[@"h1", @"heading", @"bt", @"biaoti", @"title"], @"heading1", @"H1"],
        @[@"二级标题", @"##", @[], @[@"h2", @"heading", @"bt", @"biaoti", @"title"], @"heading2", @"H2"],
        @[@"三级标题", @"###", @[], @[@"h3", @"heading", @"bt", @"biaoti", @"title"], @"heading3", @"H3"],
        @[@"四级标题", @"####", @[], @[@"h4", @"heading", @"bt", @"biaoti", @"title"], @"heading4", @"H4"],
        @[@"五级标题", @"#####", @[], @[@"h5", @"heading", @"bt", @"biaoti", @"title"], @"heading5", @"H5"],
        @[@"六级标题", @"######", @[], @[@"h6", @"heading", @"bt", @"biaoti", @"title"], @"heading6", @"H6"],
        @[@"无序列表", @"-", @[@"list.bullet"], @[@"bullet", @"ul", @"list", @"lb", @"liebiao", @"wuxu"], @"bullet"],
        @[@"有序列表", @"1.", @[@"list.number"], @[@"ordered", @"ol", @"num", @"list", @"lb", @"liebiao", @"youxu"], @"ordered"],
        @[@"任务列表", @"[ ]", @[@"checklist", @"checkmark.square"], @[@"checklist", @"todo", @"task", @"renwu", @"dbrw"], @"checklist"],
        @[@"引用", @">", @[@"text.quote"], @[@"quote", @"yinyong", @"yyl"], @"quote"],
        @[@"分割线", @"---", @[@"minus"], @[@"divider", @"hr", @"fgx", @"fenge", @"line"], @"divider"],
        @[@"代码块", @"```", @[@"chevron.left.forwardslash.chevron.right", @"curlybraces"], @[@"code", @"daima", @"dmk"], @"code"],
        @[@"公式块", @"$$", @[@"x.squareroot", @"function"], @[@"math", @"gs", @"gongshi", @"formula"], @"math"],
        @[@"表格", @"| | |", @[@"tablecells", @"square.grid.2x2"], @[@"table", @"bg", @"biaoge"], @"table"],
        @[@"分页符", @"page", @[@"rectangle.split.1x2"], @[@"pagebreak", @"page", @"fy", @"fenyefu", @"break"], @"pagebreak"],
        @[@"目录", @"[TOC]", @[@"list.bullet.indent"], @[@"toc", @"ml", @"mulu", @"catalog"], @"toc"],
        @[@"脚注", @"[^]", @[@"note.text", @"doc.plaintext"], @[@"footnote", @"jz", @"jiaozhu"], @"footnote:insert"],
        @[@"图片", @"![]()", @[@"photo"], @[@"image", @"img", @"tp", @"tupian", @"picture"], @"image"],
        @[@"提示卡片", @"[!NOTE]", @[@"info.circle"], @[@"note", @"ts", @"tishi", @"callout"], @"callout:NOTE"],
        @[@"建议卡片", @"[!TIP]", @[@"lightbulb"], @[@"tip", @"jy", @"jianyi", @"callout"], @"callout:TIP"],
        @[@"重要卡片", @"[!IMPORTANT]", @[@"exclamationmark.bubble"], @[@"important", @"zy", @"zhongyao", @"callout"], @"callout:IMPORTANT"],
        @[@"警告卡片", @"[!WARNING]", @[@"exclamationmark.triangle"], @[@"warning", @"jg", @"jinggao", @"callout"], @"callout:WARNING"],
        @[@"注意卡片", @"[!CAUTION]", @[@"exclamationmark.octagon"], @[@"caution", @"zysx", @"zhuyi", @"callout"], @"callout:CAUTION"],
    ];
    NSMutableArray *items = [NSMutableArray array];
    for (NSArray *entry in raw) {
        NSString *command = entry[4];
        [items addObject:@{@"title": entry[0], @"subtitle": entry[1], @"icons": entry[2], @"keywords": entry[3],
                           @"command": command, @"glyph": entry.count > 5 ? entry[5] : @"",
                           @"section": [self slashSectionForCommand:command]}];
    }
    return items;
}

// slashSectionForCommand groups the palette like Notion's basic/media/advanced
// headings, so a long list stays scannable.
- (NSString *)slashSectionForCommand:(NSString *)command {
    if ([command hasPrefix:@"heading"]) return @"标题";
    if ([command isEqual:@"bullet"] || [command isEqual:@"ordered"] || [command isEqual:@"checklist"]) return @"列表";
    if ([command hasPrefix:@"callout:"]) return @"提示卡片";
    return @"块";
}

// slashFiltered matches the query against titles and keywords.
- (NSArray *)slashFiltered:(NSString *)filter {
    if (!filter.length) return _slashItems;
    NSString *query = filter.lowercaseString;
    NSMutableArray *result = [NSMutableArray array];
    for (NSDictionary *item in _slashItems) {
        NSString *title = item[@"title"];
        if ([title.lowercaseString containsString:query]) { [result addObject:item]; continue; }
        BOOL hit = NO;
        for (NSString *keyword in item[@"keywords"]) {
            if ([keyword containsString:query] || [query containsString:keyword]) { hit = YES; break; }
        }
        if (hit) [result addObject:item];
    }
    return result;
}

// slashQueryRange locates a "/query" the caret sits at the end of: the whole
// paragraph content is "/" followed by non-space characters. Returns
// {NSNotFound, 0} anywhere else — code, tables, figures and non-body blocks
// never host the menu.
- (NSRange)slashQueryRange {
    if (!_text.editable || _text.hasMarkedText) return NSMakeRange(NSNotFound, 0);
    NSRange selection = _text.selectedRange;
    if (selection.length) return NSMakeRange(NSNotFound, 0);
    NSString *string = _text.string;
    NSUInteger caret = MIN(selection.location, string.length);
    NSRange paragraph = [string paragraphRangeForRange:NSMakeRange(caret, 0)];
    NSRange content = CRContent(string, paragraph);
    // The caret must sit at the end of the line: moving it into the middle of
    // "/query" dismisses the menu instead of filtering a prefix.
    if (!content.length || caret != NSMaxRange(content)) return NSMakeRange(NSNotFound, 0);
    if ([string characterAtIndex:content.location] != '/') return NSMakeRange(NSNotFound, 0);
    NSDictionary *p = CRParagraphAt(self, content.location);
    if (!CRKind(p, @"body") || CRCellAt(self, content.location) || [p[@"footnoteId"] length]) return NSMakeRange(NSNotFound, 0);
    NSRange query = NSMakeRange(content.location, caret - content.location);
    if (query.length > 24) return NSMakeRange(NSNotFound, 0);
    for (NSUInteger i = 0; i < query.length; i++) {
        unichar c = [string characterAtIndex:content.location + i];
        if (c == ' ' || c == '\t' || c == 0x3000 || c == 0xA0) return NSMakeRange(NSNotFound, 0);
    }
    return query;
}

// slashQueryChanged re-evaluates the trigger after edits and selection moves.
// It is cheap enough to run on every caret movement while the panel is closed.
- (void)slashQueryChanged {
    if (_suppress || _text.hasMarkedText) return;
    NSRange query = [self slashQueryRange];
    if (query.location == NSNotFound) { [self slashClose]; return; }
    NSString *filter = [_text.string substringWithRange:NSMakeRange(query.location + 1, query.length - 1)];
    [self slashOpenWithFilter:filter];
}

- (void)slashOpenWithFilter:(NSString *)filter {
    [self formatBarHide];
    [self emojiMenuClose];
    [self linkMenuClose];
    if (!filter) filter = @"";
    if (_slashFilter != filter && ![_slashFilter isEqualToString:filter]) {
        _slashFilter = [filter copy];
        _slashSelected = 0;
    }
    if (!_slashItems) _slashItems = [self slashBuildItems];
    NSArray *items = [self slashFiltered:_slashFilter];
    if (!_slashPanel) {
        CRSlashMenuView *view = [[CRSlashMenuView alloc] initWithOwner:self];
        CRSlashPanel *panel = [[CRSlashPanel alloc] initWithContentRect:NSMakeRect(0, 0, CRSlashWidth, 120)
                                                              styleMask:NSWindowStyleMaskBorderless | NSWindowStyleMaskNonactivatingPanel
                                                                backing:NSBackingStoreBuffered defer:NO];
        panel.releasedWhenClosed = NO;
        panel.opaque = NO;
        panel.backgroundColor = NSColor.clearColor;
        panel.hasShadow = YES;
        panel.alphaValue = 0;
        panel.contentView = view;
        _slashPanel = panel;
        _slashView = view;
        // Reposition (or dismiss) while the parent scrolls or resizes.
        _scroll.contentView.postsBoundsChangedNotifications = YES;
        _scroll.postsFrameChangedNotifications = YES;
        [[NSNotificationCenter defaultCenter] addObserver:self selector:@selector(slashParentChanged:) name:NSViewBoundsDidChangeNotification object:_scroll.contentView];
        [[NSNotificationCenter defaultCenter] addObserver:self selector:@selector(slashParentChanged:) name:NSViewFrameDidChangeNotification object:_scroll];
        // Losing key focus (another window, another app) dismisses the panel,
        // so it never floats over a window that no longer owns the editor.
        [[NSNotificationCenter defaultCenter] addObserver:self selector:@selector(slashClose) name:NSWindowDidResignKeyNotification object:_window];
        [[NSNotificationCenter defaultCenter] addObserver:self selector:@selector(slashClose) name:NSApplicationDidResignActiveNotification object:nil];
        [_window addChildWindow:panel ordered:NSWindowAbove];
    }
    _slashPanel.appearance = [NSAppearance appearanceNamed:_dark ? NSAppearanceNameDarkAqua : NSAppearanceNameAqua];
    _slashView.items = items;
    if (_slashSelected >= (NSInteger)items.count) _slashSelected = 0;
    _slashView.selected = _slashSelected;
    _slashView.hovered = -1;
    [_slashPanel setContentSize:_slashView.bounds.size];
    [_slashPanel orderFront:nil];
    [self slashPositionPanel];
    if (_slashPanel.alphaValue < 1) {
        [NSAnimationContext runAnimationGroup:^(NSAnimationContext *context) {
            context.duration = 0.09;
            _slashPanel.animator.alphaValue = 1;
        }];
    }
}

- (void)slashClose {
    if (!_slashPanel) return;
    [[NSNotificationCenter defaultCenter] removeObserver:self name:NSViewBoundsDidChangeNotification object:_scroll.contentView];
    [[NSNotificationCenter defaultCenter] removeObserver:self name:NSViewFrameDidChangeNotification object:_scroll];
    [[NSNotificationCenter defaultCenter] removeObserver:self name:NSWindowDidResignKeyNotification object:_window];
    [[NSNotificationCenter defaultCenter] removeObserver:self name:NSApplicationDidResignActiveNotification object:nil];
    [_window removeChildWindow:_slashPanel];
    [_slashPanel close];
    _slashPanel = nil;
    _slashView = nil;
    _slashFilter = nil;
    _slashSelected = 0;
}

// slashParentChanged keeps the panel glued to the caret while the document
// scrolls or the window resizes; a caret that scrolls out of sight dismisses.
- (void)slashParentChanged:(NSNotification *)note {
    if (!_slashPanel) return;
    NSRange query = [self slashQueryRange];
    if (query.location == NSNotFound) { [self slashClose]; return; }
    NSLayoutManager *layout = _text.layoutManager;
    [layout ensureLayoutForTextContainer:_text.textContainer];
    NSUInteger glyph = [layout glyphIndexForCharacterAtIndex:MIN(NSMaxRange(query), _text.string.length)];
    NSRect line = glyph < layout.numberOfGlyphs ? [layout lineFragmentRectForGlyphAtIndex:glyph effectiveRange:NULL] : layout.extraLineFragmentRect;
    line = NSOffsetRect(line, _text.textContainerOrigin.x, _text.textContainerOrigin.y);
    if (!NSIntersectsRect(line, _text.visibleRect)) { [self slashClose]; return; }
    [self slashPositionPanel];
}

// slashPositionPanel anchors the panel's top left below the caret, flipping
// above when there is no room, and clamps it to the window's frame.
- (void)slashPositionPanel {
    if (!_slashPanel || !_window) return;
    NSRange query = [self slashQueryRange];
    if (query.location == NSNotFound) return;
    NSLayoutManager *layout = _text.layoutManager;
    [layout ensureLayoutForTextContainer:_text.textContainer];
    NSUInteger glyph = [layout glyphIndexForCharacterAtIndex:MIN(NSMaxRange(query), _text.string.length)];
    NSRect caret = [layout boundingRectForGlyphRange:NSMakeRange(glyph, 0) inTextContainer:_text.textContainer];
    caret = NSOffsetRect(caret, _text.textContainerOrigin.x, _text.textContainerOrigin.y);
    NSRect anchor = [_window convertRectToScreen:[_text convertRect:caret toView:nil]];
    NSRect windowFrame = _window.frame;
    CGFloat width = _slashView.bounds.size.width, height = _slashView.bounds.size.height;
    CGFloat x = MAX(NSMinX(windowFrame) + 4, MIN(anchor.origin.x, NSMaxX(windowFrame) - width - 4));
    CGFloat y = NSMinY(anchor) - height - 6;
    if (y < NSMinY(windowFrame) + 4) y = NSMaxY(anchor) + 6;
    y = MAX(NSMinY(windowFrame) + 4, MIN(y, NSMaxY(windowFrame) - height - 4));
    [_slashPanel setFrame:NSMakeRect(x, y, width, height) display:YES];
}

// slashHandleKey routes navigation keys to the open menu. Printable
// characters fall through to normal insertion, which updates the filter.
- (BOOL)slashHandleKey:(NSEvent *)event {
    if (!_slashPanel || _text.hasMarkedText) return NO;
    NSEventModifierFlags flags = event.modifierFlags & NSEventModifierFlagDeviceIndependentFlagsMask;
    flags &= ~(NSEventModifierFlagCapsLock | NSEventModifierFlagNumericPad | NSEventModifierFlagFunction);
    if (flags) return NO;
    NSString *key = event.charactersIgnoringModifiers;
    if (key.length != 1) return NO;
    unichar c = [key characterAtIndex:0];
    if (c == 0xF700) { [self slashMove:-1]; return YES; }
    if (c == 0xF701) { [self slashMove:1]; return YES; }
    if (c == '\r' || c == 0x03) { [self slashConfirm]; return YES; }
    if (c == 0x1B) { [self slashClose]; return YES; }
    return NO;
}

- (void)slashMove:(NSInteger)delta {
    NSInteger count = (NSInteger)_slashView.items.count;
    if (!count) return;
    _slashSelected = (NSInteger)(_slashSelected + delta + count) % count;
    _slashView.selected = _slashSelected;
}

- (void)slashConfirm {
    NSArray *items = _slashView.items;
    if (_slashSelected >= 0 && _slashSelected < (NSInteger)items.count) [self slashChoose:items[_slashSelected]];
}

// slashChoose deletes the typed "/query", then applies the block in one undo
// group, through the same transactions as the toolbar commands.
- (void)slashChoose:(NSDictionary *)item {
    if (!item || !_slashPanel) return;
    [self slashClose];
    if (!_text.editable || _text.hasMarkedText) return;
    [self beginStructuralCommand];
    [_undo beginUndoGrouping];
    NSRange query = [self slashQueryRange];
    if (query.location != NSNotFound && query.length) {
        NSDictionary *body = CRBody();
        [self replace:query with:[NSAttributedString new] selection:NSMakeRange(query.location, 0)];
        _text.typingAttributes = [self visual:@{CRParagraph: body} paragraph:body];
        _emptyParagraph = body;
        [self updateTypingParagraph];
    }
    [self slashApplyCommand:item[@"command"]];
    [_undo endUndoGrouping];
    [_text scrollRangeToVisible:_text.selectedRange];
}

// slashChooseIndex drives the menu from tests, like a click would.
- (void)slashChooseIndex:(NSInteger)index {
    if (!_slashPanel) return;
    NSArray *items = _slashView.items;
    if (index < 0 || index >= (NSInteger)items.count) return;
    [self slashChoose:items[index]];
}

// slashApplyCommand maps a palette entry onto an existing editor transaction.
- (void)slashApplyCommand:(NSString *)command {
    if (![command isKindOfClass:NSString.class] || !command.length) return;
    if ([command hasPrefix:@"callout:"]) { [self setQuoteAlert:[command substringFromIndex:8]]; return; }
    if ([command isEqual:@"divider"]) { [self insertBlock:@"---"]; return; }
    if ([command isEqual:@"code"]) { [self insertBlock:@"```"]; return; }
    if ([command isEqual:@"math"]) { [self insertBlock:@"$$"]; return; }
    if ([command isEqual:@"table"]) { [self insertBlock:@"|  |  |"]; return; }
    if ([command isEqual:@"toc"]) { [self insertBlock:@"[TOC]"]; return; }
    if ([command isEqual:@"pagebreak"]) { [self insertMarkdownText:@"<div style=\"page-break-after: always\"></div>"]; return; }
    if ([command isEqual:@"image"]) { [self insertMarkdownText:@"![图片]()"]; return; }
    [self format:command]; // heading1..6, bullet, ordered, checklist, quote, footnote:insert
}

// slashSnapshotTo renders the palette to a PNG for visual checks (the panel
// lives in its own window, so the editor screenshot does not include it).
- (BOOL)slashSnapshotTo:(NSString *)path {
    if (!_slashView) return NO;
    NSRect bounds = _slashView.bounds;
    NSBitmapImageRep *rep = [_slashView bitmapImageRepForCachingDisplayInRect:bounds];
    if (!rep) return NO;
    [_slashView cacheDisplayInRect:bounds toBitmapImageRep:rep];
    NSData *png = [rep representationUsingType:NSBitmapImageFileTypePNG properties:@{}];
    return png && [png writeToFile:path atomically:YES];
}

// slashFilterText lets the palette highlight the matched part of a title.
- (NSString *)slashFilterText {
    return _slashFilter ?: @"";
}

// slashState exposes the panel to the Go test harness.
- (NSDictionary *)slashState {
    NSMutableArray *titles = [NSMutableArray array];
    for (NSDictionary *item in _slashView.items ?: @[]) [titles addObject:item[@"title"] ?: @""];
    return @{@"open": (_slashPanel ? @YES : @NO), @"filter": _slashFilter ?: @"", @"selected": @(_slashSelected), @"offset": @(_slashView.offset), @"items": titles};
}

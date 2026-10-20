// Hover affordances inside the text view: a Notion-style six-dot handle in the
// left margin that opens the block menu, and a copy button on the top-right of
// a hovered code block. Included inside @implementation CRTextView.

#pragma once

// MARK: - Hover state

// updateHoverAtPoint recomputes which affordance the pointer is over. The
// margin handle needs only the vertical position, so the caret lookup is aimed
// at the text column; the copy button only appears over a code chip.
- (void)updateHoverAtPoint:(NSPoint)point {
    CREditor *e = self.owner;
    NSRange code = NSMakeRange(NSNotFound, 0), block = NSMakeRange(NSNotFound, 0);
    NSRect chip = NSZeroRect, copy = NSZeroRect, handle = NSZeroRect, language = NSZeroRect;
    NSRange languageBlock = NSMakeRange(NSNotFound, 0);
    NSRange image = NSMakeRange(NSNotFound, 0);
    NSRect pill = NSZeroRect;
    NSArray *imageButtons = nil;
    NSRect imageCopy = NSZeroRect, imageDelete = NSZeroRect, imageReplace = NSZeroRect;
    NSTextTable *table = nil;
    NSRect tableRect = NSZeroRect;
    NSArray *tableButtons = nil;
    if (e && self.editable && !self.hasMarkedText) {
        CGFloat zoom = e.zoom;
        // A point right of the line end resolves to the text length; clamp it
        // so the last character still names its block.
        NSUInteger index = self.string.length ? MIN([self characterIndexForInsertionAtPoint:point], self.string.length - 1) : NSNotFound;
        if (index != NSNotFound) {
            NSDictionary *codeBlock = [e codeBlockAt:index];
            if (codeBlock) {
                NSRect bounds = [self codeChipRectForBlock:codeBlock];
                if (bounds.size.height > 0 && NSPointInRect(point, bounds)) {
                    code = [codeBlock[@"range"] rangeValue];
                    chip = bounds;
                    copy = NSMakeRect(NSMaxX(bounds) - 10 * zoom - 18 * zoom, NSMinY(bounds) + 7 * zoom, 18 * zoom, 18 * zoom);
                    // The language tag becomes a picker while the block is
                    // hovered; it sits left of the copy button.
                    NSString *name = [codeBlock[@"language"] length] ? codeBlock[@"language"] : @"text";
                    NSDictionary *attrs = @{NSFontAttributeName: [NSFont systemFontOfSize:10.5 * zoom]};
                    CGFloat width = [name sizeWithAttributes:attrs].width;
                    language = NSMakeRect(NSMaxX(bounds) - 10 * zoom - width - 34 * zoom - 3 * zoom,
                                          NSMinY(bounds) + 4 * zoom, width + 6 * zoom, 17 * zoom);
                    languageBlock = [codeBlock[@"range"] rangeValue];
                }
            }
        }
        // An image figure raises a size pill on hover.
        if (self.string.length) {
            for (NSValue *value in [e imageFigureRanges]) {
                NSRange figure = [value rangeValue];
                NSRange content = CRContent(self.string, figure);
                NSRange glyph = [self.layoutManager glyphRangeForCharacterRange:content actualCharacterRange:NULL];
                if (!glyph.length) continue;
                NSRect rect = [self.layoutManager boundingRectForGlyphRange:glyph inTextContainer:self.textContainer];
                rect = NSOffsetRect(rect, self.textContainerOrigin.x, self.textContainerOrigin.y);
                if (rect.size.height <= 0) continue;
                NSArray *sizes = @[@"", @"25%", @"50%", @"75%", @"100%"];
                CGFloat buttonWidth = 38 * zoom, buttonHeight = 24 * zoom, gap = 2 * zoom, pad = 5 * zoom;
                CGFloat iconWidth = 24 * zoom, iconGap = 2 * zoom, dividerGap = 9 * zoom;
                CGFloat sizesWidth = sizes.count * buttonWidth + (sizes.count - 1) * gap;
                CGFloat pillWidth = sizesWidth + 2 * pad + dividerGap + 3 * iconWidth + 2 * iconGap;
                CGFloat pillHeight = buttonHeight + 2 * pad;
                NSRect candidate = NSMakeRect(NSMaxX(rect) - pillWidth - 8 * zoom, NSMaxY(rect) - pillHeight - 8 * zoom, pillWidth, pillHeight);
                if (candidate.origin.x < NSMinX(rect) + 4 * zoom) candidate.origin.x = NSMinX(rect) + 4 * zoom;
                if (candidate.origin.y < NSMinY(rect) + 4 * zoom) candidate.origin.y = NSMaxY(rect) + 6 * zoom;
                if (!NSPointInRect(point, rect) && !NSPointInRect(point, candidate)) continue;
                image = figure;
                pill = candidate;
                NSMutableArray *buttons = [NSMutableArray array];
                for (NSUInteger i = 0; i < sizes.count; i++) {
                    [buttons addObject:[NSValue valueWithRect:NSMakeRect(NSMinX(candidate) + pad + i * (buttonWidth + gap),
                                                                        NSMinY(candidate) + pad, buttonWidth, buttonHeight)]];
                }
                imageButtons = buttons;
                CGFloat iconsX = NSMinX(candidate) + pad + sizesWidth + dividerGap;
                imageReplace = NSMakeRect(iconsX, NSMinY(candidate) + pad, iconWidth, buttonHeight);
                imageCopy = NSMakeRect(iconsX + iconWidth + iconGap, NSMinY(candidate) + pad, iconWidth, buttonHeight);
                imageDelete = NSMakeRect(iconsX + 2 * (iconWidth + iconGap), NSMinY(candidate) + pad, iconWidth, buttonHeight);
                break;
            }
        }
        // A footnote reference raises a readable card. Caret lookup rounds to
        // the nearest boundary, so the mark under the pointer is found through
        // the glyph and both of its neighbouring characters.
        NSString *footnote = nil;
        NSRect footnoteRect = NSZeroRect;
        NSDictionary *footnoteRun = [self footnoteRunAtPoint:point rect:&footnoteRect];
        if (footnoteRun) footnote = footnoteRun[@"footnote"];
        if (![footnote isEqualToString:_hoverFootnote] || !NSEqualRects(footnoteRect, _footnoteRect)) {
            _hoverFootnote = footnote;
            _footnoteRect = footnoteRect;
            [self updateFootnoteCard];
        }
        // A table raises add-row and add-column buttons on hover.
        if (index != NSNotFound) {
            NSTextTableBlock *cell = CRCellAt(self.owner, index);
            if (cell && cell.table) {
                NSRange tableRange = CRTableRange(self.owner, cell.table);
                if (tableRange.location != NSNotFound) {
                    NSRange glyph = [self.layoutManager glyphRangeForCharacterRange:tableRange actualCharacterRange:NULL];
                    if (glyph.length) {
                        NSRect bounds = [self.layoutManager boundingRectForGlyphRange:glyph inTextContainer:self.textContainer];
                        bounds = NSOffsetRect(bounds, self.textContainerOrigin.x, self.textContainerOrigin.y);
                        tableRect = NSMakeRect(self.textContainerOrigin.x, bounds.origin.y,
                                               self.textContainer.size.width, bounds.size.height);
                        CGFloat size = 20 * zoom;
                        NSRect rowButton = NSMakeRect(NSMidX(tableRect) - size / 2, NSMaxY(tableRect) + 6 * zoom, size, size);
                        NSRect columnButton = NSMakeRect(NSMaxX(tableRect) + 6 * zoom, NSMidY(tableRect) - size / 2, size, size);
                        if (NSPointInRect(point, NSInsetRect(tableRect, -12 * zoom, -12 * zoom)) ||
                            NSPointInRect(point, rowButton) || NSPointInRect(point, columnButton)) {
                            table = cell.table;
                            tableButtons = @[@{@"rect": [NSValue valueWithRect:rowButton], @"kind": @0},
                                             @{@"rect": [NSValue valueWithRect:columnButton], @"kind": @1}];
                        }
                    }
                }
            }
        }
        CGFloat originX = self.textContainerOrigin.x;
        if (point.x >= originX - 26 * zoom && point.x <= originX - 2 * zoom && self.string.length) {
            NSUInteger hit = [self characterIndexForInsertionAtPoint:NSMakePoint(originX + 2, point.y)];
            if (hit < self.string.length) {
                NSRange candidate = [e structuralBlockRangeAt:hit];
                if (candidate.location != NSNotFound && candidate.length) {
                    block = candidate;
                    NSRect line = [self lineRectForCharacter:hit];
                    handle = NSMakeRect(originX - 24 * zoom, NSMidY(line) - 9 * zoom, 18 * zoom, 18 * zoom);
                }
            }
        }
    }
    BOOL changed = !NSEqualRanges(code, _hoverCodeBlock) || !NSEqualRects(chip, _hoverCodeChip) ||
                   !NSEqualRanges(block, _hoverBlock) || !NSEqualRects(handle, _handleRect) ||
                   !NSEqualRanges(languageBlock, _languageRange) || !NSEqualRects(language, _languageRect) ||
                   !NSEqualRanges(image, _hoverImage) || !NSEqualRects(pill, _imagePill) ||
                   table != _hoverTable || !NSEqualRects(tableRect, _tableRect) ||
                   !NSEqualRects(imageCopy, _imageCopyRect) || !NSEqualRects(imageDelete, _imageDeleteRect) ||
                   !NSEqualRects(imageReplace, _imageReplaceRect);
    _hoverPoint = point;
    if (!changed) return;
    NSRect oldCopy = _copyRect, oldHandle = _handleRect, oldChip = _hoverCodeChip;
    _hoverCodeBlock = code;
    _hoverCodeChip = chip;
    _copyRect = copy;
    _hoverBlock = block;
    _handleRect = handle;
    _languageRange = languageBlock;
    _languageRect = language;
    _hoverImage = image;
    _imagePill = pill;
    _imageButtons = imageButtons;
    _imageCopyRect = imageCopy;
    _imageDeleteRect = imageDelete;
    _imageReplaceRect = imageReplace;
    _hoverTable = table;
    _tableRect = tableRect;
    _tableButtons = tableButtons;
    (void)oldCopy; (void)oldHandle; (void)oldChip;
    [self setNeedsDisplay:YES];
    [self.window invalidateCursorRectsForView:self];
}

- (void)clearHover {
    if (_hoverCodeBlock.location == NSNotFound && _hoverBlock.location == NSNotFound && _handleRect.size.height == 0) return;
    _hoverCodeBlock = NSMakeRange(NSNotFound, 0);
    _hoverBlock = NSMakeRange(NSNotFound, 0);
    _hoverCodeChip = NSZeroRect;
    _copyRect = NSZeroRect;
    _handleRect = NSZeroRect;
    _languageRange = NSMakeRange(NSNotFound, 0);
    _languageRect = NSZeroRect;
    _hoverImage = NSMakeRange(NSNotFound, 0);
    _imagePill = NSZeroRect;
    _imageButtons = nil;
    _imageCopyRect = NSZeroRect;
    _imageDeleteRect = NSZeroRect;
    _imageReplaceRect = NSZeroRect;
    _hoverTable = nil;
    _tableRect = NSZeroRect;
    _tableButtons = nil;
    _hoverFootnote = nil;
    [self hideFootnoteCard];
    [self setNeedsDisplay:YES];
    [self.window invalidateCursorRectsForView:self];
}

// lineRectForCharacter is the used line rect of index, in view coordinates.
- (NSRect)lineRectForCharacter:(NSUInteger)index {
    NSLayoutManager *layout = self.layoutManager;
    NSRange glyph = [layout glyphRangeForCharacterRange:NSMakeRange(index, 1) actualCharacterRange:NULL];
    NSRect line = glyph.length ? [layout lineFragmentUsedRectForGlyphAtIndex:glyph.location effectiveRange:NULL]
                               : layout.extraLineFragmentRect;
    return NSOffsetRect(line, self.textContainerOrigin.x, self.textContainerOrigin.y);
}

// codeChipRectForBlock mirrors drawRect's slab geometry for a code block.
- (NSRect)codeChipRectForBlock:(NSDictionary *)block {
    NSRange range = [block[@"range"] rangeValue];
    NSLayoutManager *layout = self.layoutManager;
    [layout ensureLayoutForTextContainer:self.textContainer];
    NSRange glyph = [layout glyphRangeForCharacterRange:range actualCharacterRange:NULL];
    if (!glyph.length) return NSZeroRect;
    NSRect text = [layout boundingRectForGlyphRange:glyph inTextContainer:self.textContainer];
    if (text.size.height <= 0) return NSZeroRect;
    CGFloat zoom = self.owner.zoom, pad = 12 * zoom;
    NSDictionary *paragraph = CRParagraphAt(self.owner, range.location);
    CGFloat indent = CRCodeListIndent(paragraph) * zoom;
    CGFloat inset = self.textContainer.lineFragmentPadding;
    return NSMakeRect(self.textContainerOrigin.x + inset + indent,
                      text.origin.y + self.textContainerOrigin.y - pad,
                      self.textContainer.size.width - 2 * inset - indent,
                      text.size.height + 2 * pad);
}

// MARK: - Block drag

// dragSourceIndex finds the dragged block in the current block list.
- (NSInteger)dragSourceIndexInBlocks:(NSArray *)blocks {
    for (NSInteger i = 0; i < (NSInteger)blocks.count; i++) {
        if (NSEqualRanges([blocks[i] rangeValue], _dragSource)) return i;
    }
    return -1;
}

// beginBlockDragAtPoint tracks the pointer: a click opens the block menu, a
// drag reorders the block with a live insertion line.
- (void)beginBlockDragAtPoint:(NSPoint)point {
    NSRange source = _hoverBlock;
    if (source.location == NSNotFound) return;
    _dragSource = source;
    _dragInsertion = NSNotFound;
    _dragActive = NO;
    NSPoint start = point;
    BOOL moved = NO;
    while (YES) {
        NSEvent *event = [self.window nextEventMatchingMask:NSEventMaskLeftMouseDragged | NSEventMaskLeftMouseUp
                                                   untilDate:NSDate.distantFuture
                                                       inMode:NSEventTrackingRunLoopMode dequeue:YES];
        if (!event) break;
        if (event.type == NSEventTypeLeftMouseUp) break;
        NSPoint current = [self convertPoint:event.locationInWindow fromView:nil];
        if (!moved && hypot(current.x - start.x, current.y - start.y) < 4) continue;
        moved = YES;
        _dragActive = YES;
        [self updateDragTargetAtPoint:current];
    }
    if (moved) [self commitBlockDrag];
    else [self showBlockMenuAtPoint:point];
    [self clearDrag];
}

// updateDragTargetAtPoint turns the pointer into an insertion index between
// blocks; the upper/lower half of the block under the pointer picks the side.
- (void)updateDragTargetAtPoint:(NSPoint)point {
    NSArray *blocks = [self.owner structuralBlocks];
    if (!blocks.count) return;
    // Decide by vertical position rather than caret lookup: a pointer above
    // the first block or below the last has no character under it.
    NSInteger index = -1;
    BOOL after = NO;
    for (NSInteger i = 0; i < (NSInteger)blocks.count; i++) {
        NSRect rect = [self.owner blockRectForRange:[blocks[i] rangeValue]];
        if (rect.size.height <= 0) continue;
        if (point.y < NSMinY(rect)) { index = i; after = NO; break; }
        if (point.y <= NSMaxY(rect)) { index = i; after = point.y > NSMidY(rect); break; }
    }
    if (index < 0) { index = (NSInteger)blocks.count - 1; after = YES; }
    _dragInsertion = index + (after ? 1 : 0);
    [self setNeedsDisplay:YES];
}

// commitBlockDrag walks the block across its neighbours with the same
// adjacent swap the menu and shortcuts use, inside one undo group.
- (void)commitBlockDrag {
    NSArray *blocks = [self.owner structuralBlocks];
    NSInteger source = [self dragSourceIndexInBlocks:blocks];
    if (source < 0 || _dragInsertion == NSNotFound) return;
    NSInteger insertion = _dragInsertion;
    if (insertion == source || insertion == source + 1) return;
    BOOL down = insertion > source;
    NSInteger steps = down ? insertion - source - 1 : source - insertion;
    if (steps <= 0) return;
    [self.owner beginStructuralCommand];
    [self.owner.undo beginUndoGrouping];
    self.selectedRange = NSMakeRange(_dragSource.location, 0);
    for (NSInteger i = 0; i < steps; i++) {
        if (![self.owner moveBlock:down]) break;
    }
    [self.owner.undo endUndoGrouping];
    [self setNeedsDisplay:YES];
}

// prepareTestDragTo puts the drag overlay on screen without committing, so a
// screenshot can show the insertion line.
- (void)prepareTestDragTo:(NSPoint)point {
    NSRange source = _hoverBlock;
    if (source.location == NSNotFound) return;
    _dragSource = source;
    _dragActive = YES;
    [self updateDragTargetAtPoint:point];
    [self setNeedsDisplay:YES];
}

// performTestDragTo drives the drag without the event loop, so tests are not
// at the mercy of the physical mouse position.
- (void)performTestDragTo:(NSPoint)point {
    NSRange source = _hoverBlock;
    if (source.location == NSNotFound) return;
    _dragSource = source;
    _dragActive = YES;
    [self updateDragTargetAtPoint:point];
    [self commitBlockDrag];
    [self clearDrag];
}

- (void)clearDrag {
    _dragSource = NSMakeRange(NSNotFound, 0);
    _dragInsertion = NSNotFound;
    _dragActive = NO;
    [self setNeedsDisplay:YES];
}

// MARK: - Drawing

- (void)drawDragOverlay {
    if (!_dragActive) return;
    CGFloat zoom = self.owner.zoom;
    NSColor *accent = self.owner.dark ? [NSColor colorWithRed:0.45 green:0.66 blue:1 alpha:1]
                                      : [NSColor colorWithRed:0.12 green:0.44 blue:0.92 alpha:1];
    NSRect source = [self.owner blockRectForRange:_dragSource];
    if (source.size.height > 0) {
        [[accent colorWithAlphaComponent:0.06] setFill];
        [[NSBezierPath bezierPathWithRoundedRect:source xRadius:5 * zoom yRadius:5 * zoom] fill];
        NSBezierPath *outline = [NSBezierPath bezierPathWithRoundedRect:NSInsetRect(source, 0.5, 0.5) xRadius:5 * zoom yRadius:5 * zoom];
        outline.lineWidth = 1;
        [[accent colorWithAlphaComponent:0.35] setStroke];
        [outline stroke];
    }
    NSArray *blocks = [self.owner structuralBlocks];
    NSInteger sourceIndex = [self dragSourceIndexInBlocks:blocks];
    if (_dragInsertion == NSNotFound || _dragInsertion == sourceIndex || _dragInsertion == sourceIndex + 1) return;
    NSUInteger gap = self.string.length;
    if (_dragInsertion < (NSInteger)blocks.count) gap = [blocks[_dragInsertion] rangeValue].location;
    CGFloat y;
    if (gap >= self.string.length && self.string.length) {
        NSRect last = [self lineRectForCharacter:self.string.length - 1];
        y = NSMaxY(last);
    } else {
        y = NSMinY([self lineRectForCharacter:MIN(gap, MAX(0, (NSInteger)self.string.length - 1))]);
        if (!self.string.length) y = self.textContainerOrigin.y;
    }
    CGFloat x0 = self.textContainerOrigin.x + 4 * zoom;
    CGFloat width = self.textContainer.size.width - 2 * self.textContainer.lineFragmentPadding - 8 * zoom;
    NSRect bar = NSMakeRect(x0 + 6 * zoom, y - 1.5 * zoom, width, 3 * zoom);
    [accent setFill];
    [[NSBezierPath bezierPathWithRoundedRect:bar xRadius:1.5 * zoom yRadius:1.5 * zoom] fill];
    [[NSBezierPath bezierPathWithOvalInRect:NSMakeRect(x0 - 4 * zoom, NSMidY(bar) - 3 * zoom, 6 * zoom, 6 * zoom)] fill];
}

// drawTableControls paints the add-row / add-column circles above the table.
- (void)drawTableControls {
    if (!_tableButtons.count) return;
    CGFloat zoom = self.owner.zoom;
    BOOL dark = self.owner.dark;
    NSColor *muted = CRMutedColor(dark);
    for (NSDictionary *button in _tableButtons) {
        NSRect rect = [button[@"rect"] rectValue];
        BOOL hovered = NSPointInRect(_hoverPoint, rect);
        [[(dark ? [NSColor colorWithWhite:0.22 alpha:1] : [NSColor whiteColor]) colorWithAlphaComponent:0.98] setFill];
        NSBezierPath *circle = [NSBezierPath bezierPathWithOvalInRect:rect];
        [circle fill];
        [[(dark ? [NSColor whiteColor] : [NSColor blackColor]) colorWithAlphaComponent:hovered ? 0.30 : 0.16] setStroke];
        circle.lineWidth = 1;
        [circle stroke];
        NSImage *plus = CRTintedSymbol(@"plus", 10 * zoom, NSFontWeightSemibold, hovered ? NSColor.controlAccentColor : muted);
        if (plus) {
            NSSize size = plus.size;
            [plus drawInRect:NSMakeRect(NSMidX(rect) - size.width / 2, NSMidY(rect) - size.height / 2, size.width, size.height)
                    fromRect:NSZeroRect operation:NSCompositingOperationSourceOver fraction:1 respectFlipped:YES hints:nil];
        }
    }
}

- (void)drawHoverControls {
    CGFloat zoom = self.owner.zoom;
    NSColor *muted = CRMutedColor(self.owner.dark);
    BOOL dark = self.owner.dark;
    NSColor *ink = dark ? [NSColor colorWithWhite:0.92 alpha:1] : [NSColor colorWithWhite:0.16 alpha:1];
    if (_handleRect.size.height > 0) {
        if (NSPointInRect(_hoverPoint, _handleRect)) {
            [[(dark ? [NSColor whiteColor] : [NSColor blackColor]) colorWithAlphaComponent:0.07] setFill];
            [[NSBezierPath bezierPathWithRoundedRect:_handleRect xRadius:5 * zoom yRadius:5 * zoom] fill];
        }
        [muted setFill];
        CGFloat radius = 1.4 * zoom, dx = 4.4 * zoom, dy = 5 * zoom;
        for (NSInteger row = 0; row < 3; row++) {
            for (NSInteger column = 0; column < 2; column++) {
                CGFloat x = NSMidX(_handleRect) + (column ? dx / 2 : -dx / 2);
                CGFloat y = NSMidY(_handleRect) + (row - 1) * dy;
                [[NSBezierPath bezierPathWithOvalInRect:NSMakeRect(x - radius, y - radius, radius * 2, radius * 2)] fill];
            }
        }
    }
    if (_languageRect.size.height > 0 && _languageRange.location != NSNotFound) {
        if (NSPointInRect(_hoverPoint, _languageRect)) {
            [[(dark ? [NSColor whiteColor] : [NSColor blackColor]) colorWithAlphaComponent:0.08] setFill];
            [[NSBezierPath bezierPathWithRoundedRect:_languageRect xRadius:3 * zoom yRadius:3 * zoom] fill];
        }
        NSImage *chevron = CRTintedSymbol(@"chevron.down", 8, NSFontWeightSemibold, muted);
        if (chevron) {
            NSSize size = chevron.size;
            [chevron drawInRect:NSMakeRect(NSMaxX(_languageRect) + 2 * zoom, NSMidY(_languageRect) - size.height / 2, size.width, size.height)
                      fromRect:NSZeroRect operation:NSCompositingOperationSourceOver fraction:1 respectFlipped:YES hints:nil];
        }
    }
    if (_copyRect.size.height > 0) {
        BOOL hovered = NSPointInRect(_hoverPoint, _copyRect);
        [[(dark ? [NSColor whiteColor] : [NSColor blackColor]) colorWithAlphaComponent:hovered ? 0.12 : 0.07] setFill];
        [[NSBezierPath bezierPathWithRoundedRect:_copyRect xRadius:4 * zoom yRadius:4 * zoom] fill];
        NSColor *tint = _copied ? NSColor.controlAccentColor : muted;
        NSImage *symbol = CRTintedSymbol(_copied ? @"checkmark" : @"doc.on.doc", 11 * zoom, NSFontWeightMedium, tint);
        if (symbol) {
            NSSize size = symbol.size;
            [symbol drawInRect:NSMakeRect(NSMidX(_copyRect) - size.width / 2, NSMidY(_copyRect) - size.height / 2, size.width, size.height)
                      fromRect:NSZeroRect operation:NSCompositingOperationSourceOver fraction:1 respectFlipped:YES hints:nil];
        }
    }
}

// MARK: - Actions

// hoveredCodeBlock resolves the hovered range to its block dictionary.
- (NSDictionary *)hoveredCodeBlock {
    if (_hoverCodeBlock.location == NSNotFound) return nil;
    for (NSDictionary *block in [self.owner codeBlocksIn:_hoverCodeBlock]) {
        if (NSEqualRanges([block[@"range"] rangeValue], _hoverCodeBlock)) return block;
    }
    return nil;
}

// hoveredCodeText is the block's code without fences (fences are metadata).
- (NSString *)hoveredCodeText {
    NSDictionary *block = [self hoveredCodeBlock];
    if (!block) return @"";
    NSMutableArray *lines = [NSMutableArray array];
    for (NSValue *value in block[@"lines"]) {
        NSRange range = [value rangeValue];
        [lines addObject:[self.string substringWithRange:CRContent(self.string, range)]];
    }
    while (lines.count && ![lines.lastObject length]) [lines removeLastObject];
    return [lines componentsJoinedByString:@"\n"];
}

// copyHoveredCode copies the code block's lines and flashes a checkmark.
- (BOOL)copyHoveredCode {
    if (_hoverCodeBlock.location == NSNotFound) return NO;
    NSString *code = [self hoveredCodeText];
    if (!code) return NO;
    NSPasteboard *pasteboard = NSPasteboard.generalPasteboard;
    [pasteboard clearContents];
    [pasteboard setString:code forType:NSPasteboardTypeString];
    NSUInteger token = ++_copyToken;
    _copied = YES;
    [self setNeedsDisplayInRect:NSInsetRect(_copyRect, -6, -6)];
    dispatch_after(dispatch_time(DISPATCH_TIME_NOW, (int64_t)(0.9 * NSEC_PER_SEC)), dispatch_get_main_queue(), ^{
        if (self->_copyToken != token) return;
        self->_copied = NO;
        [self setNeedsDisplayInRect:NSInsetRect(self->_copyRect, -6, -6)];
    });
    return YES;
}

// prepareBlockAtPoint selects the handle's block so the menu commands and the
// block shortcuts act on it.
- (BOOL)prepareBlockAtPoint:(NSPoint)point {
    if (_hoverBlock.location == NSNotFound) return NO;
    if (!NSPointInRect(point, _handleRect)) return NO;
    [self.window makeFirstResponder:self];
    self.selectedRange = NSMakeRange(_hoverBlock.location, 0);
    return YES;
}

// MARK: - Footnote card

// footnoteRunAtPoint finds a footnote mark whose glyph is under the pointer.
- (NSDictionary *)footnoteRunAtPoint:(NSPoint)point rect:(NSRect *)outRect {
    NSLayoutManager *layout = self.layoutManager;
    [layout ensureLayoutForTextContainer:self.textContainer];
    NSPoint container = NSMakePoint(point.x - self.textContainerOrigin.x, point.y - self.textContainerOrigin.y);
    NSUInteger glyph = [layout glyphIndexForPoint:container inTextContainer:self.textContainer];
    if (glyph >= layout.numberOfGlyphs) return nil;
    NSUInteger index = [layout characterIndexForGlyphAtIndex:glyph];
    for (NSInteger delta = 0; delta <= 1; delta++) {
        NSInteger at = (NSInteger)index - delta;
        if (at < 0 || at >= (NSInteger)self.string.length) continue;
        NSDictionary *run = [self.textStorage attribute:CRFootnote atIndex:(NSUInteger)at effectiveRange:NULL];
        if (!run) continue;
        NSRange mark = [layout glyphRangeForCharacterRange:NSMakeRange((NSUInteger)at, 1) actualCharacterRange:NULL];
        if (!mark.length) continue;
        NSRect rect = [layout boundingRectForGlyphRange:mark inTextContainer:self.textContainer];
        rect = NSOffsetRect(rect, self.textContainerOrigin.x, self.textContainerOrigin.y);
        if (!NSPointInRect(point, NSInsetRect(rect, -3, -3))) continue;
        if (outRect) *outRect = rect;
        return run;
    }
    return nil;
}

// updateFootnoteCard shows, moves or hides the card for the hovered reference.
- (void)updateFootnoteCard {
    if (!_hoverFootnote.length) {
        [self hideFootnoteCard];
        return;
    }
    if (!_footnotePanel) {
        CRFootnoteCardView *view = [[CRFootnoteCardView alloc] initWithFrame:NSMakeRect(0, 0, CRFootnoteCardWidth, 80)];
        view.owner = self.owner;
        CRFootnoteCardPanel *panel = [[CRFootnoteCardPanel alloc] initWithContentRect:NSMakeRect(0, 0, CRFootnoteCardWidth, 80)
                                                                          styleMask:NSWindowStyleMaskBorderless | NSWindowStyleMaskNonactivatingPanel
                                                                            backing:NSBackingStoreBuffered defer:NO];
        panel.releasedWhenClosed = NO;
        panel.opaque = NO;
        panel.backgroundColor = NSColor.clearColor;
        panel.hasShadow = YES;
        panel.alphaValue = 0;
        panel.ignoresMouseEvents = YES;
        panel.contentView = view;
        _footnotePanel = panel;
        _footnoteCard = view;
        [self.window addChildWindow:panel ordered:NSWindowAbove];
    }
    _footnotePanel.appearance = [NSAppearance appearanceNamed:self.owner.dark ? NSAppearanceNameDarkAqua : NSAppearanceNameAqua];
    NSDictionary *presentation = self.owner.footnotes[_hoverFootnote];
    NSString *number = presentation[@"number"];
    _footnoteCard.heading = number.length ? [NSString stringWithFormat:@"脚注 %@", number] : [NSString stringWithFormat:@"脚注 %@", _hoverFootnote];
    _footnoteCard.body = presentation[@"tooltip"] ?: @"未找到脚注定义";
    CGFloat height = [_footnoteCard cardHeight];
    [_footnotePanel setContentSize:NSMakeSize(CRFootnoteCardWidth, height)];
    NSRect anchor = [self.window convertRectToScreen:[self convertRect:_footnoteRect toView:nil]];
    NSRect windowFrame = self.window.frame;
    CGFloat x = anchor.origin.x + anchor.size.width / 2 - CRFootnoteCardWidth / 2;
    x = MAX(NSMinX(windowFrame) + 6, MIN(x, NSMaxX(windowFrame) - CRFootnoteCardWidth - 6));
    CGFloat y = NSMaxY(anchor) + 8;
    if (y + height > NSMaxY(windowFrame) - 6) y = NSMinY(anchor) - height - 8;
    y = MAX(NSMinY(windowFrame) + 6, MIN(y, NSMaxY(windowFrame) - height - 6));
    [_footnotePanel setFrame:NSMakeRect(x, y, CRFootnoteCardWidth, height) display:YES];
    [_footnotePanel orderFront:nil];
    if (_footnotePanel.alphaValue < 1) {
        [NSAnimationContext runAnimationGroup:^(NSAnimationContext *context) {
            context.duration = 0.09;
            self->_footnotePanel.animator.alphaValue = 1;
        }];
    }
}

- (void)hideFootnoteCard {
    if (!_footnotePanel) return;
    [_footnotePanel orderOut:nil];
}

- (void)closeFootnoteCard {
    if (!_footnotePanel) return;
    [self.window removeChildWindow:_footnotePanel];
    [_footnotePanel close];
    _footnotePanel = nil;
    _footnoteCard = nil;
}

- (NSDictionary *)footnoteCardState {
    return @{@"open": (_footnotePanel.isVisible ? @YES : @NO),
             @"heading": _footnoteCard.heading ?: @"",
             @"body": _footnoteCard.body ?: @""};
}

- (BOOL)footnoteCardSnapshotTo:(NSString *)path {
    if (!_footnoteCard) return NO;
    NSRect bounds = _footnoteCard.bounds;
    NSBitmapImageRep *rep = [_footnoteCard bitmapImageRepForCachingDisplayInRect:bounds];
    if (!rep) return NO;
    [_footnoteCard cacheDisplayInRect:bounds toBitmapImageRep:rep];
    NSData *png = [rep representationUsingType:NSBitmapImageFileTypePNG properties:@{}];
    return png && [png writeToFile:path atomically:YES];
}

// MARK: - Table controls

// hoveredTableCell finds the last cell in a row or column, which the table
// commands then operate on.
- (NSRange)cellRangeInTable:(NSTextTable *)table lastRow:(BOOL)lastRow {
    __block NSRange best = NSMakeRange(NSNotFound, 0);
    __block NSInteger bestRow = -1, bestColumn = -1;
    [self.textStorage enumerateAttribute:CRTableCell inRange:NSMakeRange(0, self.string.length) options:0 usingBlock:^(NSTextTableBlock *block, NSRange range, BOOL *stop) {
        if (![block isKindOfClass:NSTextTableBlock.class] || block.table != table) return;
        NSInteger row = block.startingRow, column = block.startingColumn;
        if (lastRow ? (row > bestRow || (row == bestRow && column > bestColumn))
                    : (column > bestColumn || (column == bestColumn && row > bestRow))) {
            best = range;
            bestRow = row;
            bestColumn = column;
        }
    }];
    return best;
}

// activateTableControl adds a row (0) or a column (1) at the table's end.
- (BOOL)activateTableControl:(NSInteger)kind {
    if (!_hoverTable) return NO;
    NSRange cell = [self cellRangeInTable:_hoverTable lastRow:kind == 0];
    if (cell.location == NSNotFound) return NO;
    self.selectedRange = NSMakeRange(cell.location, 0);
    [self.owner tableCommand:kind == 0 ? @"table:row-below" : @"table:column-right"];
    [self clearHover];
    return YES;
}

// MARK: - Image size pill

// drawImagePill paints the size presets on top of the image attachment.
- (void)drawImagePill {
    if (!_imagePill.size.height || !_imageButtons.count) return;
    CGFloat zoom = self.owner.zoom;
    BOOL dark = self.owner.dark;
    NSColor *muted = CRMutedColor(dark);
    NSColor *ink = dark ? [NSColor colorWithWhite:0.92 alpha:1] : [NSColor colorWithWhite:0.16 alpha:1];
    {
        [[(dark ? [NSColor colorWithWhite:0.20 alpha:1] : [NSColor whiteColor]) colorWithAlphaComponent:0.97] setFill];
        NSBezierPath *pillPath = [NSBezierPath bezierPathWithRoundedRect:_imagePill xRadius:8 * zoom yRadius:8 * zoom];
        [pillPath fill];
        [[(dark ? [NSColor whiteColor] : [NSColor blackColor]) colorWithAlphaComponent:0.14] setStroke];
        pillPath.lineWidth = 1;
        [pillPath stroke];
        NSArray *labels = @[@"原图", @"25%", @"50%", @"75%", @"100%"];
        NSString *current = [self.owner imageSpecAt:_hoverImage.location] ?: @"";
        for (NSUInteger i = 0; i < _imageButtons.count; i++) {
            NSRect button = [_imageButtons[i] rectValue];
            BOOL active = [labels[i] isEqual:@"原图"] ? !current.length : [current isEqual:labels[i]];
            if (active) {
                [[(dark ? [NSColor whiteColor] : [NSColor blackColor]) colorWithAlphaComponent:0.10] setFill];
                [[NSBezierPath bezierPathWithRoundedRect:button xRadius:5 * zoom yRadius:5 * zoom] fill];
            } else if (NSPointInRect(_hoverPoint, button)) {
                [[(dark ? [NSColor whiteColor] : [NSColor blackColor]) colorWithAlphaComponent:0.06] setFill];
                [[NSBezierPath bezierPathWithRoundedRect:button xRadius:5 * zoom yRadius:5 * zoom] fill];
            }
            NSDictionary *attrs = @{NSFontAttributeName: [NSFont systemFontOfSize:10.5 * zoom],
                                    NSForegroundColorAttributeName: active ? ink : muted};
            NSSize size = [labels[i] sizeWithAttributes:attrs];
            [labels[i] drawAtPoint:NSMakePoint(NSMidX(button) - size.width / 2, NSMidY(button) - size.height / 2) withAttributes:attrs];
        }
        // A divider separates the size presets from replace, copy and delete.
        CGFloat dividerX = NSMinX(_imageReplaceRect) - 4.5 * zoom;
        [[(dark ? [NSColor whiteColor] : [NSColor blackColor]) colorWithAlphaComponent:0.12] setFill];
        NSRectFill(NSMakeRect(dividerX, NSMinY(_imagePill) + 7 * zoom, 1, _imagePill.size.height - 14 * zoom));
        NSImage *replaceIcon = CRTintedSymbol(@"arrow.triangle.2.circlepath", 12 * zoom, NSFontWeightRegular, muted);
        NSImage *copyIcon = CRTintedSymbol(@"doc.on.doc", 12 * zoom, NSFontWeightRegular, muted);
        NSImage *deleteIcon = CRTintedSymbol(@"trash", 12 * zoom, NSFontWeightRegular,
                                             NSPointInRect(_hoverPoint, _imageDeleteRect) ? NSColor.systemRedColor : muted);
        for (NSUInteger i = 0; i < 3; i++) {
            NSRect button = i == 0 ? _imageReplaceRect : (i == 1 ? _imageCopyRect : _imageDeleteRect);
            NSImage *icon = i == 0 ? replaceIcon : (i == 1 ? copyIcon : deleteIcon);
            if (NSPointInRect(_hoverPoint, button)) {
                [[(dark ? [NSColor whiteColor] : [NSColor blackColor]) colorWithAlphaComponent:0.08] setFill];
                [[NSBezierPath bezierPathWithRoundedRect:button xRadius:5 * zoom yRadius:5 * zoom] fill];
            }
            if (!icon) continue;
            NSSize size = icon.size;
            [icon drawInRect:NSMakeRect(NSMidX(button) - size.width / 2, NSMidY(button) - size.height / 2, size.width, size.height)
                    fromRect:NSZeroRect operation:NSCompositingOperationSourceOver fraction:1 respectFlipped:YES hints:nil];
        }
    }
}

- (NSInteger)imageButtonAtPoint:(NSPoint)point {
    for (NSUInteger i = 0; i < _imageButtons.count; i++) {
        if (NSPointInRect(point, [_imageButtons[i] rectValue])) return (NSInteger)i;
    }
    return -1;
}

// imagePillButtonRect exposes a pill button to the test bridge: 0..4 are the
// size presets, 5 copy, 6 delete. Replace stays out: its click opens a panel.
- (NSRect)imagePillButtonRect:(NSInteger)index {
    if (index >= 0 && index < (NSInteger)_imageButtons.count) return [_imageButtons[index] rectValue];
    if (index == 5) return _imageCopyRect;
    if (index == 6) return _imageDeleteRect;
    return NSZeroRect;
}

// tableButtonRectForKind exposes the add-row (0) / add-column (1) button of
// the hovered table to the test bridge.
- (NSRect)tableButtonRectForKind:(NSInteger)kind {
    for (NSDictionary *button in _tableButtons) {
        if ([button[@"kind"] integerValue] == kind) return [button[@"rect"] rectValue];
    }
    return NSZeroRect;
}

// imageAttachmentInRange finds the rendered attachment inside a figure range.
- (NSTextAttachment *)imageAttachmentInRange:(NSRange)range {
    __block NSTextAttachment *found = nil;
    [self.textStorage enumerateAttribute:NSAttachmentAttributeName inRange:range options:0 usingBlock:^(NSTextAttachment *attachment, NSRange r, BOOL *stop) {
        if (attachment.image) { found = attachment; *stop = YES; }
    }];
    return found;
}

- (BOOL)copyImageAtHover {
    if (_hoverImage.location == NSNotFound) return NO;
    NSTextAttachment *attachment = [self imageAttachmentInRange:_hoverImage];
    NSImage *image = attachment.image;
    if (!image) return NO;
    NSPasteboard *pasteboard = [NSPasteboard generalPasteboard];
    [pasteboard clearContents];
    return [pasteboard writeObjects:@[image]];
}

// replaceImageAtHover asks for a new picture and swaps the figure's URL,
// keeping alt, title and size. A file inside the document's folder is written
// relative, so the note survives being moved with its assets.
- (void)replaceImageAtHover {
    if (_hoverImage.location == NSNotFound) return;
    NSUInteger location = _hoverImage.location;
    NSOpenPanel *panel = [NSOpenPanel openPanel];
    panel.title = @"替换图片";
    panel.allowedContentTypes = @[UTTypeImage];
    panel.allowsMultipleSelection = NO;
    NSUInteger revision = self.owner.documentRevision;
    NSString *snapshot = [self.string copy];
    [panel beginSheetModalForWindow:self.window completionHandler:^(NSModalResponse response) {
        if (response != NSModalResponseOK || !self.window) return;
        if (self.owner.documentRevision != revision || ![self.string isEqual:snapshot]) return;
        NSString *path = panel.URL.path;
        if (path.length) [self.owner replaceImageAtLocation:location withURL:path];
    }];
    [self clearHover];
}

- (BOOL)deleteImageAtHover {
    if (_hoverImage.location == NSNotFound) return NO;
    NSTextAttachment *attachment = [self imageAttachmentInRange:_hoverImage];
    if (!attachment) return NO;
    __block NSUInteger location = NSNotFound;
    [self.textStorage enumerateAttribute:NSAttachmentAttributeName inRange:_hoverImage options:0 usingBlock:^(id value, NSRange range, BOOL *stop) {
        if (value == attachment) { location = range.location; *stop = YES; }
    }];
    if (location == NSNotFound) return NO;
    NSRange block = [self.owner structuralBlockRangeAt:location];
    if (block.location == NSNotFound || !block.length) return NO;
    self.selectedRange = NSMakeRange(block.location, 0);
    BOOL deleted = [self.owner deleteBlock];
    [self clearHover];
    return deleted;
}

- (BOOL)applyImageSizeAtIndex:(NSInteger)index {
    if (_hoverImage.location == NSNotFound || index < 0 || index >= 5) return NO;
    NSArray *sizes = @[@"", @"25%", @"50%", @"75%", @"100%"];
    [self.owner setImageSize:sizes[index] atLocation:_hoverImage.location];
    [self clearHover];
    return YES;
}

// MARK: - Code language picker

- (void)showLanguageMenu {
    if (_languageRange.location == NSNotFound) return;
    NSRange block = _languageRange;
    NSMenu *menu = [NSMenu new];
    for (NSString *language in @[@"text", @"go", @"python", @"javascript", @"typescript", @"swift",
                                 @"rust", @"json", @"bash", @"sql", @"html", @"css", @"markdown", @"yaml"]) {
        NSMenuItem *item = [[NSMenuItem alloc] initWithTitle:language action:@selector(pickCodeLanguage:) keyEquivalent:@""];
        item.target = self;
        item.representedObject = @{@"language": language, @"range": [NSValue valueWithRange:block]};
        [menu addItem:item];
    }
    [menu addItem:NSMenuItem.separatorItem];
    NSMenuItem *custom = [[NSMenuItem alloc] initWithTitle:@"自定义…" action:@selector(chooseCodeLanguage:) keyEquivalent:@""];
    custom.target = self;
    custom.representedObject = @{@"location": @(block.location), @"revision": @(self.owner.documentRevision)};
    [menu addItem:custom];
    [menu popUpMenuPositioningItem:nil atLocation:NSMakePoint(NSMinX(_languageRect), NSMinY(_languageRect) - 2) inView:self];
    [self clearHover];
}

- (void)pickCodeLanguage:(NSMenuItem *)item {
    NSDictionary *target = item.representedObject;
    if (![target isKindOfClass:NSDictionary.class]) return;
    NSRange range = [target[@"range"] rangeValue];
    [self.owner setCodeLanguage:target[@"language"] atLocation:range.location];
    [self clearHover];
}

- (void)showBlockMenuAtPoint:(NSPoint)point {
    // Capture the geometry and block first: selecting the block clears the
    // hover state, which the popup position and the commands depend on.
    NSRect handle = _handleRect;
    NSRange block = _hoverBlock;
    if (block.location == NSNotFound || !NSPointInRect(point, handle)) return;
    [self.window makeFirstResponder:self];
    self.selectedRange = NSMakeRange(block.location, 0);
    NSMenu *menu = [NSMenu new];
    [self appendBlockActions:menu];
    [menu popUpMenuPositioningItem:nil atLocation:NSMakePoint(NSMidX(handle), NSMinY(handle) - 2) inView:self];
    [self clearHover];
}

// hoverState exposes the affordances to the Go test harness.
- (NSDictionary *)hoverState {
    return @{
        @"copy": (_copyRect.size.height > 0 ? @YES : @NO),
        @"handle": (_handleRect.size.height > 0 ? @YES : @NO),
        @"language": (_languageRect.size.height > 0 ? @YES : @NO),
        @"image": (_hoverImage.location != NSNotFound ? @YES : @NO),
        @"table": (_hoverTable ? @YES : @NO),
        @"block": NSStringFromRange(_hoverBlock),
        @"copyRect": @{@"x": @(_copyRect.origin.x), @"y": @(_copyRect.origin.y), @"w": @(_copyRect.size.width), @"h": @(_copyRect.size.height)},
        @"handleRect": @{@"x": @(_handleRect.origin.x), @"y": @(_handleRect.origin.y), @"w": @(_handleRect.size.width), @"h": @(_handleRect.size.height)},
    };
}

// MARK: - Tracking

- (void)updateTrackingAreas {
    [super updateTrackingAreas];
    if (_hoverTracking) [self removeTrackingArea:_hoverTracking];
    _hoverTracking = [[NSTrackingArea alloc] initWithRect:self.bounds
                                                  options:NSTrackingMouseMoved | NSTrackingMouseEnteredAndExited |
                                                          NSTrackingActiveInKeyWindow | NSTrackingInVisibleRect
                                                    owner:self userInfo:nil];
    [self addTrackingArea:_hoverTracking];
    self.window.acceptsMouseMovedEvents = YES;
}
- (void)mouseMoved:(NSEvent *)event {
    [super mouseMoved:event];
    [self updateHoverAtPoint:[self convertPoint:event.locationInWindow fromView:nil]];
}
- (void)mouseExited:(NSEvent *)event {
    [super mouseExited:event];
    [self clearHover];
}
- (void)resetCursorRects {
    [super resetCursorRects];
    if (_copyRect.size.height > 0) [self addCursorRect:_copyRect cursor:NSCursor.pointingHandCursor];
    if (_languageRect.size.height > 0) [self addCursorRect:_languageRect cursor:NSCursor.pointingHandCursor];
    for (NSValue *value in _imageButtons) [self addCursorRect:[value rectValue] cursor:NSCursor.pointingHandCursor];
    if (_imageReplaceRect.size.width) [self addCursorRect:_imageReplaceRect cursor:NSCursor.pointingHandCursor];
    if (_imageCopyRect.size.width) [self addCursorRect:_imageCopyRect cursor:NSCursor.pointingHandCursor];
    if (_imageDeleteRect.size.width) [self addCursorRect:_imageDeleteRect cursor:NSCursor.pointingHandCursor];
    for (NSDictionary *button in _tableButtons) [self addCursorRect:[button[@"rect"] rectValue] cursor:NSCursor.pointingHandCursor];
    if (_handleRect.size.height > 0) [self addCursorRect:_handleRect cursor:NSCursor.pointingHandCursor];
}

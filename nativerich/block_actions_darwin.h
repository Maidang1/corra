// Structural block actions: move a block past its neighbour, duplicate it, or
// delete it. Blocks are the units a Markdown writer thinks in — a paragraph, a
// list item, a whole code block, a table — so a fence or table never splits.
// Included inside @implementation CREditor (after format_bar_darwin.h).

#pragma once

// MARK: - Block actions

// structuralBlockRangeAt expands location to its block: code lines join their
// whole fence, a table cell joins its whole table, other paragraphs stand
// alone.
- (NSRange)structuralBlockRangeAt:(NSUInteger)location {
    NSString *string = _text.string;
    if (!string.length) return NSMakeRange(NSNotFound, 0);
    NSRange paragraph = [string paragraphRangeForRange:NSMakeRange(MIN(location, string.length), 0)];
    NSRange block = paragraph;
    NSTextTableBlock *cell = CRCellAt(self, paragraph.location);
    if (cell) {
        NSRange table = CRTableRange(self, cell.table);
        if (table.location != NSNotFound) block = [string paragraphRangeForRange:table];
        return block;
    }
    if (CRKind(CRParagraphAt(self, paragraph.location), @"code")) {
        while (block.location > 0) {
            NSRange previous = [string paragraphRangeForRange:NSMakeRange(block.location - 1, 0)];
            if (!CRKind(CRParagraphAt(self, previous.location), @"code")) break;
            block = NSUnionRange(previous, block);
        }
        while (NSMaxRange(block) < string.length) {
            NSRange next = [string paragraphRangeForRange:NSMakeRange(NSMaxRange(block), 0)];
            if (!CRKind(CRParagraphAt(self, next.location), @"code")) break;
            block = NSUnionRange(block, next);
        }
    }
    return block;
}

// structuralBlockRange covers every selected paragraph plus the block each
// end belongs to.
- (NSRange)structuralBlockRange {
    NSArray *selected = CRSelectedParagraphs(self);
    if (!selected.count) return NSMakeRange(NSNotFound, 0);
    NSRange first = [selected.firstObject rangeValue], last = [selected.lastObject rangeValue];
    NSRange block = NSMakeRange(first.location, NSMaxRange(last) - first.location);
    NSRange head = [self structuralBlockRangeAt:first.location];
    NSRange tail = [self structuralBlockRangeAt:last.location];
    if (head.location != NSNotFound) block = NSUnionRange(block, head);
    if (tail.location != NSNotFound) block = NSUnionRange(block, tail);
    return block;
}

// structuralBlocks lists the document's blocks in order: code fences and
// tables are one unit, every other paragraph stands alone. Dragging needs the
// list to count how many neighbours a block must cross.
- (NSArray *)structuralBlocks {
    NSMutableArray *blocks = [NSMutableArray array];
    NSString *string = _text.string;
    NSUInteger at = 0;
    while (at < string.length) {
        NSRange block = [self structuralBlockRangeAt:at];
        if (block.location == NSNotFound || !block.length) break;
        [blocks addObject:[NSValue valueWithRange:block]];
        at = NSMaxRange(block);
    }
    if (!blocks.count && string.length) [blocks addObject:[NSValue valueWithRange:NSMakeRange(0, string.length)]];
    return blocks;
}

// blockRectForRange is the block's line bounds with a little padding, used to
// outline the block being dragged.
- (NSRect)blockRectForRange:(NSRange)range {
    NSLayoutManager *layout = _text.layoutManager;
    [layout ensureLayoutForTextContainer:_text.textContainer];
    if (!range.length || NSMaxRange(range) > _text.string.length) return NSZeroRect;
    NSRange glyph = [layout glyphRangeForCharacterRange:range actualCharacterRange:NULL];
    if (!glyph.length) return NSZeroRect;
    NSRect text = [layout boundingRectForGlyphRange:glyph inTextContainer:_text.textContainer];
    if (text.size.height <= 0) return NSZeroRect;
    CGFloat pad = 6 * _zoom;
    return NSMakeRect(_text.textContainerOrigin.x + _text.textContainer.lineFragmentPadding,
                      text.origin.y + _text.textContainerOrigin.y - pad,
                      _text.textContainer.size.width - 2 * _text.textContainer.lineFragmentPadding,
                      text.size.height + 2 * pad);
}

// CRBlockSeparator is the gap a block writes after itself (blank line, tight
// list break, or the end of the document).
static NSString *CRBlockSeparator(CREditor *editor, NSUInteger location) {
    NSDictionary *paragraph = CRParagraphAt(editor, location);
    return [paragraph[@"separator"] isKindOfClass:NSString.class] ? paragraph[@"separator"] : @"";
}

// CRSetBoundarySeparators keeps the two gaps of the swapped pair in place:
// the paragraph that now ends the first unit takes the gap the old first unit
// had, and the final paragraph takes the old trailing gap. Without this a
// swapped pair would either merge into one paragraph or grow a blank line at
// the end of the note.
static void CRSetBoundarySeparators(NSMutableAttributedString *text, NSUInteger firstLength, NSString *firstGap, NSString *trailingGap) {
    NSArray *ranges = CRRanges(text.string);
    NSInteger firstLast = -1, last = -1;
    for (NSUInteger i = 0; i < ranges.count; i++) {
        NSRange range = [ranges[i] rangeValue];
        if (!range.length) continue;
        if (range.location < firstLength) firstLast = (NSInteger)i;
        last = (NSInteger)i;
    }
    for (NSUInteger i = 0; i < ranges.count; i++) {
        NSRange range = [ranges[i] rangeValue];
        if (!range.length) continue;
        NSString *gap = (NSInteger)i == firstLast ? firstGap : (NSInteger)i == last ? trailingGap : nil;
        if (!gap) continue;
        NSDictionary *stored = [text attribute:CRParagraph atIndex:range.location effectiveRange:NULL];
        if (![stored isKindOfClass:NSDictionary.class]) continue;
        NSMutableDictionary *updated = [stored mutableCopy];
        if (gap.length) updated[@"separator"] = gap;
        else [updated removeObjectForKey:@"separator"];
        [text addAttribute:CRParagraph value:CRStoredParagraph(updated) range:range];
    }
}

// moveBlock swaps the current block with the one before or after it. Tables
// stay put: their cell blocks cannot be relaid out inside a reordered range.
- (BOOL)moveBlock:(BOOL)down {
    if (!_text.editable || _text.hasMarkedText || _suppress) return NO;
    NSRange block = [self structuralBlockRange];
    if (block.location == NSNotFound || !block.length) return NO;
    NSString *string = _text.string;
    if (CRCellAt(self, block.location)) return NO;
    if ([CRParagraphAt(self, block.location)[@"footnoteId"] length]) return NO;
    NSRange target;
    if (down) {
        if (NSMaxRange(block) >= string.length) return NO;
        target = [self structuralBlockRangeAt:NSMaxRange(block)];
    } else {
        if (block.location == 0) return NO;
        target = [self structuralBlockRangeAt:block.location - 1];
    }
    if (target.location == NSNotFound || !target.length) return NO;
    if (NSIntersectionRange(block, target).length) return NO;
    if (CRCellAt(self, target.location)) return NO;
    if ([CRParagraphAt(self, target.location)[@"footnoteId"] length]) return NO;
    NSRange span = NSUnionRange(block, target);
    NSRange blockLast = [string paragraphRangeForRange:NSMakeRange(NSMaxRange(block) - 1, 0)];
    NSRange targetLast = [string paragraphRangeForRange:NSMakeRange(NSMaxRange(target) - 1, 0)];
    NSString *blockGap = CRBlockSeparator(self, blockLast.location);
    NSString *targetGap = CRBlockSeparator(self, targetLast.location);
    NSAttributedString *blockText = [_text.textStorage attributedSubstringFromRange:block];
    NSAttributedString *targetText = [_text.textStorage attributedSubstringFromRange:target];
    NSMutableAttributedString *combined = [NSMutableAttributedString new];
    NSUInteger firstLength = 0;
    if (down) {
        [combined appendAttributedString:targetText];
        [combined appendAttributedString:blockText];
        firstLength = targetText.length;
    } else {
        [combined appendAttributedString:blockText];
        [combined appendAttributedString:targetText];
        firstLength = blockText.length;
    }
    // The text view stores one newline per paragraph but the note's final
    // paragraph has none. Swapping the last two blocks would glue their text
    // together, so move the break to the new boundary and drop the trailing one.
    if (NSMaxRange(span) == string.length && ![string hasSuffix:@"\n"]) {
        if (firstLength && [combined.string characterAtIndex:firstLength - 1] != '\n') {
            NSDictionary *edge = [combined attributesAtIndex:firstLength - 1 effectiveRange:NULL];
            [combined insertAttributedString:[[NSAttributedString alloc] initWithString:@"\n" attributes:edge] atIndex:firstLength];
            firstLength++;
        }
        if ([combined.string hasSuffix:@"\n"]) [combined deleteCharactersInRange:NSMakeRange(combined.length - 1, 1)];
    }
    // The gap after the new first unit is the one the old first unit had.
    NSString *firstGap = down ? blockGap : targetGap;
    NSString *trailingGap = down ? targetGap : blockGap;
    CRSetBoundarySeparators(combined, firstLength, firstGap, trailingGap);
    NSUInteger offset = _text.selectedRange.location > block.location ? _text.selectedRange.location - block.location : 0;
    NSUInteger blockStart = down ? span.location + firstLength : span.location;
    [self beginStructuralCommand];
    [self replace:span with:combined selection:NSMakeRange(MIN(blockStart + offset, span.location + combined.length), 0)];
    [self renumber];
    return YES;
}

// duplicateBlock inserts a fresh copy after the block, through the same
// fragment importer the clipboard uses (so list identities are regenerated).
- (BOOL)duplicateBlock {
    if (!_text.editable || _text.hasMarkedText || _suppress) return NO;
    NSRange block = [self structuralBlockRange];
    if (block.location == NSNotFound || !block.length) return NO;
    NSDictionary *document = [self documentForRange:block];
    NSArray *paragraphs = document[@"paragraphs"];
    if (!paragraphs.count) return NO;
    paragraphs = CRImportedParagraphs(paragraphs);
    NSData *data = [NSJSONSerialization dataWithJSONObject:@{@"paragraphs": paragraphs} options:0 error:NULL];
    if (!data) return NO;
    _text.selectedRange = NSMakeRange(MIN(NSMaxRange(block), _text.string.length), 0);
    cr_insert((uintptr_t)(__bridge void *)self, [[NSString alloc] initWithData:data encoding:NSUTF8StringEncoding].UTF8String);
    return YES;
}

// deleteBlock removes the block and keeps the document's ending intact.
- (BOOL)deleteBlock {
    if (!_text.editable || _text.hasMarkedText || _suppress) return NO;
    NSRange block = [self structuralBlockRange];
    if (block.location == NSNotFound || !block.length) return NO;
    NSDictionary *ending = _emptyParagraph;
    NSUInteger lengthBefore = _text.string.length;
    BOOL atEOF = NSMaxRange(block) >= lengthBefore;
    // At the end of the note the break before the block goes with it, or the
    // text view would keep an empty trailing paragraph.
    NSRange removal = block;
    if (atEOF && block.location > 0) removal = NSMakeRange(block.location - 1, block.length + 1);
    [self replace:removal with:[NSAttributedString new] selection:NSMakeRange(removal.location, 0)];
    // Deleting the final block must not leave its blank line behind.
    if (atEOF && _text.string.length) {
        NSRange last = [_text.string paragraphRangeForRange:NSMakeRange(_text.string.length - 1, 0)];
        NSDictionary *stored = CRParagraphAt(self, last.location);
        if ([stored[@"separator"] length]) {
            NSMutableDictionary *updated = [stored mutableCopy];
            [updated removeObjectForKey:@"separator"];
            [_text.textStorage addAttribute:CRParagraph value:CRStoredParagraph(updated) range:last];
        }
    }
    if (!_text.string.length) {
        _emptyParagraph = ending ?: CRBody();
        _text.typingAttributes = [self visual:@{CRParagraph: _emptyParagraph} paragraph:_emptyParagraph];
    } else {
        [self updateTypingParagraph];
    }
    [self renumber];
    return YES;
}

// Included in CREditor's implementation. Fence metadata bounds each block:
// neighboring code paragraphs can belong to distinct adjacent fences.
- (NSArray *)codeBlocks { return [self codeBlocksIn:NSMakeRange(0,_text.string.length)]; }
// codeBlocksIn: lists the blocks touching within, scanning only from the
// opening fence of a block it cuts into to the end of the last such block.
- (NSArray *)codeBlocksIn:(NSRange)within {
    NSString *string=_text.string;
    NSUInteger start=[string paragraphRangeForRange:NSMakeRange(MIN(within.location,string.length),0)].location;
    while (start>0) {
        NSDictionary *p=CRParagraphAt(self,start);
        if (!CRKind(p,@"code") || [p[@"open"] length]) break;
        NSUInteger previous=[string paragraphRangeForRange:NSMakeRange(start-1,0)].location;
        if (!CRKind(CRParagraphAt(self,previous),@"code")) break;
        start=previous;
    }
    NSUInteger end=MIN(NSMaxRange(within),string.length);
    if (end>start) end=NSMaxRange([string paragraphRangeForRange:NSMakeRange(end-1,0)]);
    while (end<string.length) {
        NSDictionary *p=CRParagraphAt(self,end);
        if (!CRKind(p,@"code") || [p[@"open"] length]) break;
        end=NSMaxRange([string paragraphRangeForRange:NSMakeRange(end,0)]);
    }
    NSArray *ranges=start==0 && end>=string.length ? CRRanges(string) : CRRangesIn(string,NSMakeRange(start,end>start ? end-start-1 : 0));
    NSMutableArray *blocks=[NSMutableArray array];
    for (NSUInteger i=0;i<ranges.count;) {
        NSRange first=[ranges[i] rangeValue];
        NSDictionary *p=CRParagraphAt(self,first.location);
        if (!CRKind(p,@"code")) { i++; continue; }
        NSUInteger j=i;
        NSRange whole=first;
        for (;j<ranges.count;j++) {
            NSRange line=[ranges[j] rangeValue];
            NSDictionary *q=CRParagraphAt(self,line.location);
            if (!CRKind(q,@"code") || (j>i && [q[@"open"] length])) break;
            whole=NSUnionRange(whole,line);
        }
        NSString *open=[p[@"open"] isKindOfClass:NSString.class] ? p[@"open"] : @"```";
        NSUInteger cursor=0;
        while (cursor<open.length && [[NSCharacterSet whitespaceCharacterSet] characterIsMember:[open characterAtIndex:cursor]]) cursor++;
        while (cursor<open.length && ([open characterAtIndex:cursor]=='`' || [open characterAtIndex:cursor]=='~')) cursor++;
        while (cursor<open.length && [[NSCharacterSet whitespaceCharacterSet] characterIsMember:[open characterAtIndex:cursor]]) cursor++;
        NSUInteger end=cursor;
        while (end<open.length && ![[NSCharacterSet whitespaceAndNewlineCharacterSet] characterIsMember:[open characterAtIndex:end]]) end++;
        NSString *language=[open substringWithRange:NSMakeRange(cursor,end-cursor)];
        [blocks addObject:@{@"range":[NSValue valueWithRange:whole],@"open":open,@"language":language,@"languageRange":[NSValue valueWithRange:NSMakeRange(cursor,end-cursor)],@"lines":[ranges subarrayWithRange:NSMakeRange(i,j-i)]}];
        i=MAX(i+1,j);
    }
    return blocks;
}
- (void)setCodeLanguage:(NSString *)language {
    [self setCodeLanguage:language atLocation:_text.selectedRange.location];
}
- (void)setCodeLanguage:(NSString *)language atLocation:(NSUInteger)caret {
    if (!_text.editable || _text.hasMarkedText || ![language isKindOfClass:NSString.class]) return;
    NSMutableCharacterSet *invalid=[[NSCharacterSet whitespaceAndNewlineCharacterSet] mutableCopy];
    [invalid formUnionWithCharacterSet:NSCharacterSet.controlCharacterSet];
    [invalid addCharactersInString:@"`~"];
    if ([language rangeOfCharacterFromSet:invalid].location!=NSNotFound) return;
    for (NSDictionary *block in [self codeBlocks].reverseObjectEnumerator) {
        NSRange range=[block[@"range"] rangeValue];
        if (!(NSLocationInRange(caret,range) || (caret==NSMaxRange(range) && caret==_text.string.length))) continue;
        NSString *open=block[@"open"];
        NSRange token=[block[@"languageRange"] rangeValue];
        NSString *replacement=[open stringByReplacingCharactersInRange:token withString:language];
        if ([replacement isEqual:open]) return;
        NSRange first=[block[@"lines"][0] rangeValue];
        NSMutableDictionary *paragraph=[CRParagraphAt(self,first.location) mutableCopy];
        paragraph[@"open"]=replacement;
        NSMutableAttributedString *value=[_text.textStorage mutableCopy];
        NSDictionary *ending=_emptyParagraph;
        NSMutableDictionary *typing=[_text.typingAttributes mutableCopy];
        if (first.length) [value addAttribute:CRParagraph value:paragraph range:first];
        else { ending=paragraph; typing[CRParagraph]=paragraph; }
        if (NSLocationInRange(caret,first)) typing[CRParagraph]=paragraph;
        [self beginStructuralCommand];
        [_undo beginUndoGrouping];
        [self restore:value selection:_text.selectedRange typing:typing ending:ending];
        [_undo endUndoGrouping];
        return;
    }
}
- (CGFloat)codeGutterWidth {
    return _codeLineNumbers ? _codeGutter : 0;
}
- (NSArray *)codeLineNumberEntries {
    if (!_codeLineNumbers) return @[];
    [_text.layoutManager ensureLayoutForTextContainer:_text.textContainer];
    return [self codeLineNumberEntriesIn:NSMakeRange(0,_text.string.length)];
}
- (NSArray *)codeLineNumberEntriesIn:(NSRange)within {
    if (!_codeLineNumbers) return @[];
    NSMutableArray *entries=[NSMutableArray array];
    CGFloat gutter=[self codeGutterWidth];
    NSDictionary *attributes=@{NSFontAttributeName:[NSFont monospacedDigitSystemFontOfSize:11.5*_zoom weight:NSFontWeightRegular]};
    for (NSDictionary *block in [self codeBlocks]) {
        NSRange whole=[block[@"range"] rangeValue];
        if (NSMaxRange(whole)<within.location || whole.location>NSMaxRange(within)) continue;
        NSUInteger number=1;
        for (NSValue *value in block[@"lines"]) {
            NSRange range=value.rangeValue;
            NSRect line=NSZeroRect;
            NSRect glyphBounds=NSZeroRect;
            if (range.location<_text.string.length) {
                NSUInteger glyph=[_text.layoutManager glyphIndexForCharacterAtIndex:range.location];
                NSRange fragment;
                line=[_text.layoutManager lineFragmentRectForGlyphAtIndex:glyph effectiveRange:&fragment];
                glyphBounds=[_text.layoutManager boundingRectForGlyphRange:fragment inTextContainer:_text.textContainer];
                // The fragment includes paragraph spacing before a fence's
                // first line; glyph bounds align the number with its text.
                if (glyphBounds.size.height>0) line=glyphBounds;
            } else line=_text.layoutManager.extraLineFragmentRect;
            if (line.size.height<=0) line=NSMakeRect(0,0,_text.textContainer.size.width,[_text.layoutManager defaultLineHeightForFont:[NSFont monospacedSystemFontOfSize:13*_zoom weight:NSFontWeightRegular]]);
            NSString *label=[NSString stringWithFormat:@"%lu",(unsigned long)number];
            NSSize size=[label sizeWithAttributes:attributes];
            NSRect rect=NSMakeRect(_text.textContainerOrigin.x+gutter-12*_zoom-size.width,_text.textContainerOrigin.y+line.origin.y+(line.size.height-size.height)/2,size.width,size.height);
            [entries addObject:@{@"number":@(number++),@"location":@(range.location),@"length":@(range.length),@"x":@(rect.origin.x),@"y":@(rect.origin.y),@"width":@(rect.size.width),@"height":@(rect.size.height),@"glyphY":@(_text.textContainerOrigin.y+glyphBounds.origin.y),@"glyphHeight":@(glyphBounds.size.height)}];
        }
    }
    return entries;
}
- (void)drawCodeLineNumbers:(NSRect)dirty range:(NSRange)within {
    NSDictionary *attributes=@{NSFontAttributeName:[NSFont monospacedDigitSystemFontOfSize:11.5*_zoom weight:NSFontWeightRegular],NSForegroundColorAttributeName:_dark ? [NSColor colorWithWhite:0.57 alpha:1] : [NSColor colorWithWhite:0.48 alpha:1]};
    for (NSDictionary *entry in [self codeLineNumberEntriesIn:within]) {
        NSRect rect=NSMakeRect([entry[@"x"] doubleValue],[entry[@"y"] doubleValue],[entry[@"width"] doubleValue],[entry[@"height"] doubleValue]);
        if (NSIntersectsRect(rect,dirty)) [[entry[@"number"] description] drawInRect:rect withAttributes:attributes];
    }
}
- (NSDictionary *)codeControlState {
    NSMutableArray *blocks=[NSMutableArray array];
    CGFloat indent=0;
    for (NSDictionary *block in [self codeBlocks]) {
        NSRange range=[block[@"range"] rangeValue];
        [blocks addObject:@{@"location":@(range.location),@"length":@(range.length),@"open":block[@"open"],@"language":block[@"language"]}];
        NSDictionary *attrs=range.location<_text.string.length ? [_text.textStorage attributesAtIndex:range.location effectiveRange:NULL] : _text.typingAttributes;
        indent=[attrs[NSParagraphStyleAttributeName] headIndent];
    }
    return @{@"blocks":blocks,@"lines":[self codeLineNumberEntries],@"gutter":@([self codeGutterWidth]),@"enabled":@(_codeLineNumbers),@"actualIndent":@(indent),@"insetY":@(_text.textContainerInset.height)};
}
- (void)highlightCode { [self highlightCodeInRange:NSMakeRange(0,_text.string.length)]; }
// highlightCodeInRange re-tokenizes only the code blocks touching dirty.
- (void)highlightCodeInRange:(NSRange)dirty {
    BOOL whole=dirty.location==0 && dirty.length>=_text.string.length;
    NSArray *blocks=[self codeBlocksIn:dirty];
    NSUInteger maximum=1;
    for (NSDictionary *block in blocks) maximum=MAX(maximum,[block[@"lines"] count]);
    NSString *label=[NSString stringWithFormat:@"%lu",(unsigned long)maximum];
    NSFont *font=[NSFont monospacedDigitSystemFontOfSize:11.5*_zoom weight:NSFontWeightRegular];
    // The gutter fits the longest block; a local edit can only widen it.
    CGFloat gutter=ceil([label sizeWithAttributes:@{NSFontAttributeName:font}].width+24*_zoom);
    _codeGutter=whole ? gutter : MAX(_codeGutter,gutter);
    for (NSDictionary *block in blocks) {
        NSRange whole=[block[@"range"] rangeValue];
        if (NSMaxRange(whole)<dirty.location || whole.location>NSMaxRange(dirty)) continue;
        [_text.textStorage removeAttribute:CRHl range:whole];
        NSString *language=block[@"language"];
        if (!language.length) continue;
        NSRange range=CRContent(_text.string,whole);
        if (!range.length) continue;
        char *out=cr_highlight((char *)[_text.string substringWithRange:range].UTF8String,(char *)language.UTF8String);
        if (!out) continue;
        NSArray *spans=[NSJSONSerialization JSONObjectWithData:[[NSString stringWithUTF8String:out] dataUsingEncoding:NSUTF8StringEncoding] options:0 error:NULL];
        free(out);
        NSUInteger cursor=range.location;
        for (NSDictionary *span in spans) {
            NSString *text=span[@"Text"], *cls=span[@"Class"];
            NSUInteger length=MIN(text.length,NSMaxRange(range)-cursor);
            if (length && cls.length && ![cls isEqual:@"text"]) [_text.textStorage addAttribute:CRHl value:cls range:NSMakeRange(cursor,length)];
            cursor+=length;
            if (cursor>=NSMaxRange(range)) break;
        }
    }
}

// Typora-style block input rules that fire on Enter: a thematic break, a
// display-math block, and a pipe table header. Included inside CREditor.

// CRRuleText: the line is a Markdown thematic break (---, ***, ___, - - -).
static BOOL CRRuleText(NSString *text) {
    static NSRegularExpression *rule;
    if (!rule) rule=[NSRegularExpression regularExpressionWithPattern:@"^ {0,3}([-*_])(?:[ \\t]*\\1){2,}[ \\t]*$" options:0 error:NULL];
    return [rule firstMatchInString:text options:0 range:NSMakeRange(0,text.length)]!=nil;
}

// CRTableHeader splits "| a | b |" into its cells, or nil if it is not one.
static NSArray<NSString *> *CRTableHeader(NSString *text) {
    NSString *line=[text stringByTrimmingCharactersInSet:NSCharacterSet.whitespaceCharacterSet];
    if (line.length<3 || ![line hasPrefix:@"|"] || ![line hasSuffix:@"|"]) return nil;
    NSMutableArray *cells=[NSMutableArray array];
    for (NSString *cell in [[line substringWithRange:NSMakeRange(1,line.length-2)] componentsSeparatedByString:@"|"])
        [cells addObject:[cell stringByTrimmingCharactersInSet:NSCharacterSet.whitespaceCharacterSet]];
    return cells;
}

// CRRuleActive: the caret touches the rule line, so it shows as source.
static BOOL CRRuleActive(CREditor *e, NSRange paragraph) {
    NSRange selection=e.text.selectedRange, content=CRContent(e.text.string,paragraph);
    return NSMaxRange(selection)>=content.location && selection.location<=NSMaxRange(content);
}

// blockRuleOnEnter applies a block rule to the body line ending at the caret
// and returns YES if one fired.
- (BOOL)blockRuleOnEnter:(NSRange)paragraph {
    NSRange content=CRContent(_text.string,paragraph);
    NSString *text=[_text.string substringWithRange:content];
    BOOL trailing=NSMaxRange(paragraph)>NSMaxRange(content);
    if (CRRuleText(text)) {
        // The rule keeps its own spelling; Enter continues on a body line.
        NSDictionary *rule=@{@"kind":@"source",@"runs":@[],@"separator":@"\n\n"}, *body=CRBody();
        NSDictionary *ruleAttrs=[self visual:@{CRParagraph:rule} paragraph:rule], *bodyAttrs=[self visual:@{CRParagraph:body} paragraph:body];
        NSMutableAttributedString *value=[[NSMutableAttributedString alloc] initWithString:[text stringByAppendingString:@"\n"] attributes:ruleAttrs];
        if (trailing) [value appendAttributedString:[[NSAttributedString alloc] initWithString:@"\n" attributes:bodyAttrs]];
        [self replace:paragraph with:value selection:NSMakeRange(paragraph.location+text.length+1,0)];
        _text.typingAttributes=bodyAttrs;
        if (!trailing) _emptyParagraph=body;
        [self restyle];
        return YES;
    }
    if ([text isEqual:@"$$"]) {
        // Open the block with its closing fence; it renders once the caret
        // leaves, like a revealed formula.
        NSDictionary *source=@{@"kind":@"source",@"raw":@"$$\n\n$$",@"separator":@"\n\n"};
        NSMutableAttributedString *value=[[self sourceText:source] mutableCopy];
        NSDictionary *last=[value attributesAtIndex:value.length-1 effectiveRange:NULL];
        if (trailing) [value appendAttributedString:[[NSAttributedString alloc] initWithString:@"\n" attributes:last]];
        NSDictionary *ending=_emptyParagraph;
        [self replace:paragraph with:value selection:NSMakeRange(paragraph.location+3,0)];
        if (trailing) _emptyParagraph=ending;
        [self updateTypingParagraph];
        _draftGroup=[self draftGroupAt:_text.selectedRange.location];
        return YES;
    }
    NSArray *header=CRTableHeader(text);
    if (header.count) {
        NSMutableArray *head=[NSMutableArray array], *row=[NSMutableArray array], *alignments=[NSMutableArray array];
        for (NSString *cell in header) {
            [head addObject:@{@"runs":cell.length ? @[@{@"text":cell}] : @[]}];
            [row addObject:@{@"runs":@[]}];
            [alignments addObject:@"none"];
        }
        NSDictionary *table=@{@"kind":@"table",@"runs":@[],@"rows":@[head,row],@"alignments":alignments};
        NSDictionary *ending=_emptyParagraph;
        NSAttributedString *value=[self attributedDocument:@{@"paragraphs":@[table]} replacingRange:paragraph];
        _emptyParagraph=trailing ? ending : CRBody();
        // The caret lands in the first body cell, after the header cells.
        NSUInteger caret=paragraph.location, cells=0;
        while (cells<header.count && caret-paragraph.location<value.length) {
            NSRange r=[value.string paragraphRangeForRange:NSMakeRange(caret-paragraph.location,0)];
            caret=paragraph.location+NSMaxRange(r); cells++;
        }
        [self replace:paragraph with:value selection:NSMakeRange(caret,0)];
        [self updateTypingParagraph];
        return YES;
    }
    return NO;
}

// refreshRules hides the spelling of thematic breaks the caret is not on;
// drawRect paints the rule in their place. Display-only: storage unchanged.
- (void)refreshRules {
    NSLayoutManager *layout=_text.layoutManager;
    NSString *string=_text.string;
    if (!string.length) return;
    [layout removeTemporaryAttribute:NSForegroundColorAttributeName forCharacterRange:NSMakeRange(0,string.length)];
    for (NSValue *v in [self ruleRanges]) [layout addTemporaryAttribute:NSForegroundColorAttributeName value:NSColor.clearColor forCharacterRange:CRContent(string,v.rangeValue)];
    for (NSDictionary *marker in [self alertMarkers]) [layout addTemporaryAttribute:NSForegroundColorAttributeName value:NSColor.clearColor forCharacterRange:[marker[@"range"] rangeValue]];
}

// alertMarkers lists the "[!LABEL]" markers of callouts the caret is not in;
// drawRect paints an icon and title over each, GitHub style.
- (NSArray<NSDictionary *> *)alertMarkers {
    NSMutableArray *result=[NSMutableArray array];
    for (NSValue *v in CRRanges(_text.string)) {
        NSRange r=v.rangeValue, content=CRContent(_text.string,r);
        NSDictionary *p=CRParagraphAt(self,r.location);
        if (!content.length || !CRKind(p,@"quote") || CRRuleActive(self,r)) continue;
        NSString *text=[_text.string substringWithRange:content], *label=CRAlertLabelOf(text);
        if (!label) continue;
        NSUInteger length=[text rangeOfString:@"]"].location+1;
        [result addObject:@{@"range":[NSValue valueWithRange:NSMakeRange(content.location,length)],@"label":label}];
    }
    return result;
}
- (void)drawAlertTitles {
    NSDictionary *symbols=@{@"NOTE":@"info.circle",@"TIP":@"lightbulb",@"IMPORTANT":@"exclamationmark.bubble",@"WARNING":@"exclamationmark.triangle",@"CAUTION":@"exclamationmark.octagon"};
    for (NSDictionary *marker in [self alertMarkers]) {
        NSRange range=[marker[@"range"] rangeValue];
        NSRange glyph=[_text.layoutManager glyphRangeForCharacterRange:range actualCharacterRange:NULL];
        if (!glyph.length) continue;
        NSRect box=NSOffsetRect([_text.layoutManager boundingRectForGlyphRange:glyph inTextContainer:_text.textContainer],_text.textContainerOrigin.x,_text.textContainerOrigin.y);
        NSFont *base=[_text.textStorage attribute:NSFontAttributeName atIndex:range.location effectiveRange:NULL] ?: [NSFont systemFontOfSize:16*_zoom];
        NSColor *accent=CRAlertAccent(marker[@"label"],_dark);
        NSDictionary *attrs=@{NSFontAttributeName:[NSFont systemFontOfSize:base.pointSize weight:NSFontWeightSemibold],NSForegroundColorAttributeName:accent};
        NSString *title=[marker[@"label"] capitalizedString];
        CGFloat icon=round(base.pointSize*0.9), x=NSMinX(box);
        NSImage *symbol=[NSImage imageWithSystemSymbolName:symbols[marker[@"label"]] accessibilityDescription:title];
        if (symbol) {
            NSImageSymbolConfiguration *config=[NSImageSymbolConfiguration configurationWithPointSize:icon weight:NSFontWeightSemibold];
            if (@available(macOS 12.0, *)) config=[config configurationByApplyingConfiguration:[NSImageSymbolConfiguration configurationWithPaletteColors:@[accent]]];
            symbol=[symbol imageWithSymbolConfiguration:config];
            NSSize size=symbol.size;
            [symbol drawInRect:NSMakeRect(x,NSMidY(box)-size.height/2,size.width,size.height) fromRect:NSZeroRect operation:NSCompositingOperationSourceOver fraction:1 respectFlipped:YES hints:nil];
            x+=size.width+4*_zoom;
        }
        NSSize size=[title sizeWithAttributes:attrs];
        [title drawAtPoint:NSMakePoint(x,NSMidY(box)-size.height/2) withAttributes:attrs];
    }
}
- (NSArray<NSValue *> *)ruleRanges {
    // Only a source line standing alone is a rule; "---" inside a larger
    // source block (front matter, a revealed figure) stays text.
    NSMutableArray *result=[NSMutableArray array];
    NSArray *ranges=CRRanges(_text.string);
    for (NSUInteger i=0;i<ranges.count;i++) {
        NSRange r=[ranges[i] rangeValue], content=CRContent(_text.string,r);
        NSDictionary *p=CRParagraphAt(self,r.location);
        if (!content.length || !CRKind(p,@"source")) continue;
        NSDictionary *previous=i ? CRParagraphAt(self,[ranges[i-1] rangeValue].location) : nil;
        NSDictionary *next=i+1<ranges.count && [ranges[i+1] rangeValue].length ? CRParagraphAt(self,[ranges[i+1] rangeValue].location) : nil;
        if (CRKind(previous,@"source") && !CRBlockEnding(previous)) continue;
        if (CRKind(next,@"source") && !CRBlockEnding(p)) continue;
        if (CRRuleText([_text.string substringWithRange:content]) && !CRRuleActive(self,r)) [result addObject:ranges[i]];
    }
    return result;
}

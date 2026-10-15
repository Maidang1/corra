- (void)continueListParagraph {
    if (!_text.editable || _text.hasMarkedText) return;
    NSRange selection=_text.selectedRange;
    NSDictionary *original=CRParagraphAt(self,selection.location);
    if (!CRList(original)) return;
    [self beginStructuralCommand];
    NSMutableDictionary *next=[original mutableCopy];
    for (NSString *key in @[@"id",@"nextId",@"raw",@"baseline",@"listIndent",@"listBaseline",@"separator"]) [next removeObjectForKey:key];
    next[@"listContinuation"]=@YES;
    if (!next[@"listItem"]) next[@"listItem"]=NSUUID.UUID.UUIDString;
    NSAttributedString *newline=[[NSAttributedString alloc] initWithString:@"\n" attributes:[self visual:@{CRParagraph:original} paragraph:original]];
    BOOL previous=_suppress; _suppress=YES;
    [self replace:selection with:newline selection:NSMakeRange(selection.location+1,0)];
    NSRange range=[_text.string paragraphRangeForRange:NSMakeRange(selection.location+1,0)];
    if (range.length) [_text.textStorage addAttribute:CRParagraph value:next range:range];
    else _emptyParagraph=next;
    _text.typingAttributes=[self visual:@{CRParagraph:next} paragraph:next];
    [self restyle]; _suppress=previous; [self notify];
}
- (BOOL)indentQuote:(BOOL)outdent {
    NSArray *ranges=CRSelectedParagraphs(self);
    if (!ranges.count) return NO;
    for (NSValue *value in ranges) if (!CRKind(CRParagraphAt(self,value.rangeValue.location),@"quote")) return NO;
    if (!_text.editable || _text.hasMarkedText) return YES;
    [self beginStructuralCommand];
    NSMutableAttributedString *after=[_text.textStorage mutableCopy];
    NSDictionary *empty=nil;
    for (NSValue *value in ranges) {
        NSRange range=value.rangeValue;
        NSMutableDictionary *p=[CRParagraphAt(self,range.location) mutableCopy];
        NSInteger depth=[p[@"depth"] integerValue];
        if (outdent && depth==0) { p[@"kind"]=@"body"; [p removeObjectForKey:@"depth"]; }
        else p[@"depth"]=@(MAX(0,depth+(outdent ? -1 : 1)));
        if (range.length) [after addAttribute:CRParagraph value:p range:range]; else empty=p;
    }
    BOOL previous=_suppress; _suppress=YES;
    [self replace:NSMakeRange(0,_text.string.length) with:after selection:_text.selectedRange];
    if (empty) { _emptyParagraph=empty; _text.typingAttributes=[self visual:@{CRParagraph:empty} paragraph:empty]; }
    [self restyle]; [self updateTypingParagraph]; _suppress=previous; [self notify];
    return YES;
}

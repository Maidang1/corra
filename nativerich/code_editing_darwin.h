// Included in CREditor's implementation. Editing line prefixes in reverse
// keeps original ranges valid and retains the hidden fence metadata.
- (BOOL)indentCode:(BOOL)outdent {
    NSArray *ranges=CRSelectedParagraphs(self);
    if (!ranges.count) return NO;
    for (NSValue *value in ranges) if (!CRKind(CRParagraphAt(self,value.rangeValue.location),@"code")) return NO;
    if (!_text.editable || _text.hasMarkedText) return YES;
    NSMutableAttributedString *after=[_text.textStorage mutableCopy];
    NSRange selection=_text.selectedRange;
    NSInteger start=selection.location, end=NSMaxRange(selection);
    BOOL changed=NO;
    for (NSValue *value in ranges.reverseObjectEnumerator) {
        NSRange range=CRContent(_text.string,value.rangeValue);
        NSUInteger remove=0, add=outdent ? 0 : 1;
        if (outdent) {
            if (range.length && [_text.string characterAtIndex:range.location]=='\t') remove=1;
            else while (remove<MIN(4,range.length) && [_text.string characterAtIndex:range.location+remove]==' ') remove++;
            if (!remove) continue;
        }
        NSDictionary *p=CRParagraphAt(self,range.location);
        NSDictionary *attrs=range.location<_text.string.length ? [_text.textStorage attributesAtIndex:range.location effectiveRange:NULL] : [self visual:@{CRParagraph:p} paragraph:p];
        NSAttributedString *prefix=[[NSAttributedString alloc] initWithString:outdent ? @"" : @"\t" attributes:attrs];
        [after replaceCharactersInRange:NSMakeRange(range.location,remove) withAttributedString:prefix];
        NSInteger (^adjust)(NSInteger)=^NSInteger(NSInteger position) {
            if (position<(NSInteger)range.location) return position;
            if (position<=(NSInteger)(range.location+remove)) return range.location+add;
            return position+(NSInteger)add-(NSInteger)remove;
        };
        start=adjust(start); end=adjust(end); changed=YES;
    }
    if (changed) [self replace:NSMakeRange(0,_text.string.length) with:after selection:NSMakeRange(start,MAX(0,end-start))];
    return YES;
}

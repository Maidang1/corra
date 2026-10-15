// Included in CREditor's implementation. Markers are metadata, so changing
// indentation keeps all UTF-16 offsets and inline attributes intact.
- (void)indentList:(BOOL)outdent {
    if (!_text.editable || _text.hasMarkedText) return;
    NSArray *ranges=CRRanges(_text.string);
    NSArray *selected=CRSelectedParagraphs(self);
    NSMutableIndexSet *roots=[NSMutableIndexSet indexSet];
    NSMutableIndexSet *affected=[NSMutableIndexSet indexSet];
    for (NSValue *value in selected) {
        NSUInteger index=[ranges indexOfObject:value];
        NSDictionary *p=CRParagraphAt(self,value.rangeValue.location);
        if (!CRList(p) || index==NSNotFound) return;
        if ([p[@"listContinuation"] boolValue] && p[@"listItem"]) {
            for (NSInteger parent=(NSInteger)index-1; parent>=0; parent--) {
                NSDictionary *candidate=CRParagraphAt(self,[ranges[parent] rangeValue].location);
                if ([candidate[@"listItem"] isEqual:p[@"listItem"]] && ![candidate[@"listContinuation"] boolValue]) { index=parent; p=candidate; break; }
            }
        }
        if ([affected containsIndex:index]) continue;
        NSInteger depth=[p[@"depth"] integerValue];
        if (!outdent) {
            NSInteger previous=(NSInteger)index-1;
            while (previous>=0 && CRList(CRParagraphAt(self,[ranges[previous] rangeValue].location)) &&
                   [CRParagraphAt(self,[ranges[previous] rangeValue].location)[@"depth"] integerValue]>depth) previous--;
            if (previous<0) return;
            NSDictionary *parent=CRParagraphAt(self,[ranges[previous] rangeValue].location);
            if (!CRList(parent) || [parent[@"depth"] integerValue]!=depth) return;
            if (p[@"listId"] && parent[@"listId"] && ![p[@"listId"] isEqual:parent[@"listId"]]) return;
        }
        [roots addIndex:index]; [affected addIndex:index];
        for (NSUInteger child=index+1; child<ranges.count; child++) {
            NSDictionary *next=CRParagraphAt(self,[ranges[child] rangeValue].location);
            BOOL continuation=[next[@"listContinuation"] boolValue] && p[@"listItem"] && [next[@"listItem"] isEqual:p[@"listItem"]];
            if (!CRList(next) || ([next[@"depth"] integerValue]<=depth && !continuation)) break;
            [affected addIndex:child];
        }
    }
    if (!affected.count) return;
    [self beginStructuralCommand];
    NSMutableAttributedString *after=[_text.textStorage mutableCopy];
    NSMutableDictionary *destinations=[NSMutableDictionary dictionary];
    NSMutableDictionary *itemDestinations=[NSMutableDictionary dictionary];
    __block NSDictionary *empty=nil;
    [affected enumerateIndexesUsingBlock:^(NSUInteger index, BOOL *stop) {
        NSRange range=[ranges[index] rangeValue];
        NSMutableDictionary *p=[CRParagraphAt(self,range.location) mutableCopy];
        NSString *item=p[@"listItem"];
        NSInteger depth=[p[@"depth"] integerValue];
        NSDictionary *owner=item ? itemDestinations[item] : nil;
        if (outdent && depth==0 && ([roots containsIndex:index] || (owner && CRKind(owner,@"body")))) {
            p[@"kind"]=@"body";
            for (NSString *key in @[@"depth",@"ordinal",@"checked",@"listIndent",@"listId",@"listBaseline",@"listContainer",@"listMarker",@"listItem",@"listContinuation"]) [p removeObjectForKey:key];
        } else {
            NSInteger targetDepth=MAX(0,depth+(outdent ? -1 : 1));
            p[@"depth"]=@(targetDepth);
            if ([roots containsIndex:index]) {
                NSString *key=[NSString stringWithFormat:@"%@:%ld",p[@"listContainer"] ?: @"",(long)targetDepth];
                NSDictionary *destination=destinations[key];
                if (!destination) {
                    for (NSInteger previous=(NSInteger)index-1; previous>=0; previous--) {
                        NSDictionary *candidate=CRParagraphAt(self,[ranges[previous] rangeValue].location);
                        if (!CRList(candidate) || [candidate[@"depth"] integerValue]<targetDepth) break;
                        if ([candidate[@"depth"] integerValue]==targetDepth) {
                            if (CROrdered(candidate)==CROrdered(p)) destination=candidate;
                            break;
                        }
                    }
                    if (!destination) destination=@{@"listContainer":NSUUID.UUID.UUIDString,@"listMarker":p[@"listMarker"] ?: (CROrdered(p) ? @"." : @"-")};
                    destinations[key]=destination;
                }
                p[@"listContainer"]=destination[@"listContainer"] ?: NSUUID.UUID.UUIDString;
                p[@"listMarker"]=destination[@"listMarker"] ?: (CROrdered(p) ? @"." : @"-");
            } else if (owner && [p[@"listContinuation"] boolValue]) {
                p[@"listContainer"]=owner[@"listContainer"];
                p[@"listMarker"]=owner[@"listMarker"];
            }
        }
        if (item && ![p[@"listContinuation"] boolValue]) itemDestinations[item]=p;
        if (range.length) [after addAttribute:CRParagraph value:p range:range];
        else empty=p;
    }];
    NSRange selection=_text.selectedRange;
    BOOL previous=_suppress; _suppress=YES;
    [self replace:NSMakeRange(0,_text.string.length) with:after selection:selection];
    if (empty) { _emptyParagraph=empty; _text.typingAttributes=[self visual:@{CRParagraph:empty} paragraph:empty]; }
    [self renumber]; [self restyle]; [self updateTypingParagraph];
    _suppress=previous; [self notify];
}

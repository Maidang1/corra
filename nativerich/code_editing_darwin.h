// Included in CREditor's implementation. Rebuild the active fenced body when
// paragraph edges change so hidden fences survive selected text replacement.
// listContinuationAfterCode: leaving a code block that belongs to a list
// item continues that item with a paragraph (Typora), instead of ending the
// list. nil when the code is not inside a list.
- (NSDictionary *)listContinuationAfterCode:(NSDictionary *)code before:(NSUInteger)location {
    NSString *item=code[@"listItem"];
    if (!item.length) return nil;
    NSUInteger at=location;
    for (int guard=0; at>0 && guard<500; guard++) {
        NSRange previous=[_text.string paragraphRangeForRange:NSMakeRange(at-1,0)];
        NSDictionary *p=CRParagraphAt(self,previous.location);
        if (![p[@"listItem"] isEqual:item]) { if (!CRKind(p,@"code")) break; }
        else if (CRList(p)) {
            NSMutableDictionary *next=[p mutableCopy];
            for (NSString *key in @[@"id",@"nextId",@"baseline",@"raw",@"listBaseline",@"listIndent",@"separator"]) [next removeObjectForKey:key];
            next[@"listContinuation"]=@YES; next[@"runs"]=@[];
            return next;
        }
        at=previous.location;
    }
    return nil;
}
// leaveCodeBlock: Cmd+Return anywhere in a code block continues below it
// (Typora), by the same path as Return on its empty last line.
- (BOOL)leaveCodeBlock {
    NSRange selection=_text.selectedRange;
    NSString *string=_text.string;
    if (!_text.editable || _text.hasMarkedText || !CRKind(CRParagraphAt(self,selection.location),@"code")) return NO;
    NSRange line=[string paragraphRangeForRange:NSMakeRange(selection.location,0)];
    while (NSMaxRange(line)<string.length) {
        NSDictionary *next=CRParagraphAt(self,NSMaxRange(line));
        if (!CRKind(next,@"code") || [next[@"open"] length]) break;
        line=[string paragraphRangeForRange:NSMakeRange(NSMaxRange(line),0)];
    }
    NSRange content=CRContent(string,line);
    _text.selectedRange=NSMakeRange(NSMaxRange(content),0);
    if (content.length) [self codeNewline];
    [self codeNewline];
    return YES;
}
- (void)codeNewline {
    // A new code line keeps the indentation of the line it splits.
    NSRange selection=_text.selectedRange;
    NSRange line=[_text.string paragraphRangeForRange:NSMakeRange(selection.location,0)];
    NSUInteger end=line.location;
    while (end<selection.location && ([_text.string characterAtIndex:end]==' ' || [_text.string characterAtIndex:end]=='\t')) end++;
    NSString *indent=[_text.string substringWithRange:NSMakeRange(line.location,end-line.location)];
    // A whitespace-only line gets no indent, so a second Enter can leave.
    if (end==NSMaxRange(CRContent(_text.string,line))) indent=@"";
    [self replaceCodeRange:selection withText:[@"\n" stringByAppendingString:indent] newlineCommand:YES];
}
- (BOOL)replaceCodeRange:(NSRange)selection withText:(NSString *)text {
    return [self replaceCodeRange:selection withText:text newlineCommand:NO];
}
- (BOOL)replaceCodeRange:(NSRange)selection withText:(NSString *)text newlineCommand:(BOOL)newlineCommand {
    if (!_text.editable || _text.hasMarkedText || ![self allowRange:selection] || ![text isKindOfClass:NSString.class]) return NO;
    text=[[text stringByReplacingOccurrencesOfString:@"\r\n" withString:@"\n"] stringByReplacingOccurrencesOfString:@"\r" withString:@"\n"];
    BOOL touchesCode=NO;
    for (NSDictionary *block in [self codeBlocks]) {
        NSArray *lines=block[@"lines"];
        NSRange range=[block[@"range"] rangeValue];
        NSRange last=[lines.lastObject rangeValue];
        NSRange lastContent=CRContent(_text.string,last);
        NSRange body=NSMakeRange(range.location,NSMaxRange(lastContent)-range.location);
        if (NSIntersectionRange(selection,range).length || (selection.location>=body.location && selection.location<=NSMaxRange(body)) || (selection.length && NSMaxRange(selection)==body.location)) touchesCode=YES;
        if (selection.location<body.location || selection.location>NSMaxRange(body) || NSMaxRange(selection)>NSMaxRange(range)) continue;
        // Triple-click selects a paragraph's display newline as well. That
        // separator belongs to the neighboring block, not to the code body.
        NSRange edit=selection;
        if (NSMaxRange(edit)>NSMaxRange(body)) edit.length=NSMaxRange(body)-edit.location;
        NSString *source=[_text.string substringWithRange:body];
        BOOL exit=newlineCommand && edit.length==0 && edit.location==last.location && lastContent.length==0;
        NSString *changed=exit ? source : [source stringByReplacingCharactersInRange:NSMakeRange(edit.location-body.location,edit.length) withString:text];
        if (!exit && [source isEqual:changed]) {
            if (edit.length) _text.selectedRange=NSMakeRange(edit.location+text.length,0);
            return YES;
        }
        NSArray *parts=[changed componentsSeparatedByString:@"\n"];
        NSDictionary *first=CRParagraphAt(self,range.location);
        NSDictionary *final=CRParagraphAt(self,last.location);
        NSString *newline=[first[@"codeNewline"] isKindOfClass:NSString.class] ? first[@"codeNewline"] : @"";
        if (!newline.length) newline=[first[@"separator"] containsString:@"\r\n"] || [final[@"separator"] containsString:@"\r\n"] ? @"\r\n" : @"\n";
        NSString *separator=[final[@"separator"] isKindOfClass:NSString.class] ? final[@"separator"] : @"";
        if ([first[@"codeClose"] length]) separator=[first[@"codeSeparator"] isKindOfClass:NSString.class] ? first[@"codeSeparator"] : @"";
        NSString *close=@"";
        for (NSValue *value in lines.reverseObjectEnumerator) {
            NSString *candidate=CRParagraphAt(self,value.rangeValue.location)[@"close"];
            if (candidate.length) { close=candidate; break; }
        }
        if (!close.length) close=[first[@"codeClose"] isKindOfClass:NSString.class] ? first[@"codeClose"] : @"";
        if (!close.length) close=[final[@"codeClose"] isKindOfClass:NSString.class] ? final[@"codeClose"] : @"";
        if (!close.length) {
            NSString *open=block[@"open"];
            NSUInteger cursor=0;
            while (cursor<open.length && [[NSCharacterSet whitespaceCharacterSet] characterIsMember:[open characterAtIndex:cursor]]) cursor++;
            NSUInteger end=cursor;
            while (end<open.length && ([open characterAtIndex:end]=='`' || [open characterAtIndex:end]=='~')) end++;
            close=end>cursor ? [open substringWithRange:NSMakeRange(cursor,end-cursor)] : @"```";
        }
        NSMutableAttributedString *replacement=[NSMutableAttributedString new];
        NSDictionary *ending=nil, *caretParagraph=nil;
        NSUInteger codeCount=exit ? parts.count-1 : parts.count;
        for (NSUInteger i=0;i<parts.count;i++) {
            BOOL code=i<codeCount;
            NSDictionary *exitParagraph=code ? nil : [self listContinuationAfterCode:first before:range.location];
            NSMutableDictionary *p=[(code ? first : exitParagraph ?: CRInFootnote(CRBody(),first)) mutableCopy];
            for (NSString *key in @[@"id",@"nextId",@"baseline",@"raw",@"open",@"close"]) [p removeObjectForKey:key];
            if (code) {
                // A boxed C "||" expression is an int, which would serialize
                // as 0/1 and break Go's bool decoding of the whole document.
                if ([first[@"footnoteId"] length]) p[@"footnoteContinuation"]=@((BOOL)(i>0 || [first[@"footnoteContinuation"] boolValue]));
                p[@"codeNewline"]=newline;
                p[@"codeClose"]=close;
                p[@"codeSeparator"]=separator;
                p[@"separator"]=i+1==codeCount ? separator : newline;
                if (i==0) p[@"open"]=block[@"open"];
                if (i+1==codeCount) p[@"close"]=close;
            } else p[@"separator"]=separator;
            NSDictionary *attributes=[self visual:@{CRParagraph:p} paragraph:p];
            NSUInteger offset=replacement.length;
            [replacement appendAttributedString:[[NSAttributedString alloc] initWithString:parts[i] attributes:attributes]];
            if (i+1<parts.count || NSMaxRange(range)>NSMaxRange(body)) [replacement appendAttributedString:[[NSAttributedString alloc] initWithString:@"\n" attributes:attributes]];
            NSUInteger caret=exit ? source.length : edit.location-body.location+text.length;
            if (caret>=offset && (caret<replacement.length || i+1==parts.count)) caretParagraph=p;
            ending=p;
        }
        NSMutableAttributedString *after=[_text.textStorage mutableCopy];
        [after replaceCharactersInRange:range withAttributedString:replacement];
        NSMutableDictionary *typing=[_text.typingAttributes mutableCopy];
        [typing removeObjectForKey:CRHl];
        typing[CRParagraph]=caretParagraph ?: ending;
        NSUInteger caret=range.location+(exit ? source.length : edit.location-body.location+text.length);
        BOOL endingInBlock=last.location==_text.string.length || (NSMaxRange(range)==_text.string.length && ![_text.string hasSuffix:@"\n"]);
        [self beginStructuralCommand];
        [_undo beginUndoGrouping];
        [self restore:after selection:NSMakeRange(caret,0) typing:typing ending:endingInBlock ? ending : _emptyParagraph];
        [_undo endUndoGrouping];
        return YES;
    }
    // A replacement crossing a fence boundary needs a document-level
    // structural operation; raw AppKit replacement would erase its fences.
    return touchesCode;
}
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
    if (changed) { [self beginStructuralCommand]; [self replace:NSMakeRange(0,_text.string.length) with:after selection:NSMakeRange(start,MAX(0,end-start))]; }
    return YES;
}

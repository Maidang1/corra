// Included in CREditor: reveal overlapping inline formats as one editable
// fragment, retaining the native paragraph/table and its surrounding text.
- (BOOL)prepareInlineReplacement:(NSRange *)range {
    if (_suppress || _text.hasMarkedText || !range->length || NSMaxRange(*range)>_text.string.length) return NO;
    __block BOOL crossing=NO;
    [_text.textStorage enumerateAttribute:CRInlineSource inRange:*range options:0 usingBlock:^(id source, NSRange part, BOOL *stop) {
        if (!source) return;
        NSRange full;
        [self.text.textStorage attribute:CRInlineSource atIndex:part.location longestEffectiveRange:&full inRange:NSMakeRange(0,self.text.string.length)];
        if (range->location<full.location || NSMaxRange(*range)>NSMaxRange(full)) crossing=YES;
    }];
    if (!crossing) return NO;
    BOOL previous=_suppress; _suppress=YES;
    _text.selectedRange=*range;
    _suppress=previous;
    [self finishInlineSource:YES];
    *range=_text.selectedRange;
    return YES;
}
- (BOOL)revealInlineAt:(NSUInteger)location {
    if (_suppress || !_text.editable || _text.hasMarkedText || _text.selectedRange.length || location>=_text.string.length) return NO;
    if ([self revealInlineMathAt:location]) return YES;
    NSDictionary *paragraph=CRParagraphAt(self,location);
    if ([@[@"source",@"code",@"opaque",@"math",@"image",@"diagram",@"toc"] containsObject:paragraph[@"kind"]]) return NO;
    NSRange limit=CRContent(_text.string,[_text.string paragraphRangeForRange:NSMakeRange(location,0)]);
    if (!NSLocationInRange(location,limit)) return NO;
    NSDictionary *attrs=[_text.textStorage attributesAtIndex:location effectiveRange:NULL];
    if (attrs[CRInlineSource]) return NO;
    NSArray *keys=@[CRBold,CRItalic,CRCode,NSStrikethroughStyleAttributeName,CRUnderline,CRHighlight,CRSup,CRSub,NSLinkAttributeName,CRFootnote];
    BOOL styled=NO;
    for (NSString *key in keys) if (attrs[key] && ![attrs[key] isEqual:@NO]) styled=YES;
    if (!styled) return NO;
    NSRange range=NSMakeRange(location,1), previous;
    do {
        previous=range;
        for (NSString *key in keys) {
            __block NSRange expanded=range;
            [_text.textStorage enumerateAttribute:key inRange:range options:0 usingBlock:^(id value, NSRange part, BOOL *stop) {
                if (!value || [value isEqual:@NO]) return;
                NSRange effective;
                [self.text.textStorage attribute:key atIndex:part.location longestEffectiveRange:&effective inRange:limit];
                expanded=NSUnionRange(expanded,effective);
            }];
            range=expanded;
        }
    } while (!NSEqualRanges(previous,range));
    NSDictionary *document=[self documentForRange:range];
    NSArray *paragraphs=document[@"paragraphs"];
    if (paragraphs.count!=1) return NO;
    NSData *data=[NSJSONSerialization dataWithJSONObject:paragraphs[0][@"runs"] ?: @[] options:0 error:NULL];
    if (!data) return NO;
    NSString *json=[[NSString alloc] initWithData:data encoding:NSUTF8StringEncoding];
    NSTextTableBlock *cell=CRCellAt(self,range.location);
    char *result=cr_reveal_inline((char *)json.UTF8String,cell?1:0);
    if (!result) return NO;
    NSDictionary *fragment=[NSJSONSerialization JSONObjectWithData:[[NSString stringWithUTF8String:result] dataUsingEncoding:NSUTF8StringEncoding] options:0 error:NULL]; free(result);
    NSString *source=fragment[@"source"];
    NSArray *positions=fragment[@"positions"];
    NSUInteger caretLocation=_text.selectedRange.location;
    NSUInteger offset=caretLocation>=range.location && caretLocation<=NSMaxRange(range) ? caretLocation-range.location : location-range.location;
    if (!source || offset>=positions.count) return NO;
    NSUInteger caret=[positions[offset] unsignedIntegerValue];
    if (caret>source.length) return NO;
    NSDictionary *origin=@{@"id":NSUUID.UUID.UUIDString,@"source":source,@"runs":paragraphs[0][@"runs"] ?: @[]};
    NSMutableDictionary *typing=[@{CRParagraph:paragraph,CRInlineSource:origin} mutableCopy];
    if (cell) typing[CRTableCell]=cell;
    NSAttributedString *content=[[NSAttributedString alloc] initWithString:source attributes:[self visual:typing paragraph:paragraph]];
    [self replace:range with:content selection:NSMakeRange(range.location+caret,0)];
    _text.typingAttributes=[self visual:typing paragraph:paragraph];
    return YES;
}

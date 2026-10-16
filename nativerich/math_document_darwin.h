- (NSArray *)mathForDocument:(NSDictionary *)document {
    if (!_callback) return @[];
    NSData *data=[NSJSONSerialization dataWithJSONObject:document options:0 error:NULL];
    if (!data) return @[];
    NSString *source=[[NSString alloc] initWithData:data encoding:NSUTF8StringEncoding];
    char *json=cr_render_math_document(_callback,source.UTF8String,_dark?1:0,_zoom,(_mathNumbering ?: @"none").UTF8String);
    if (!json) return @[];
    NSString *value=[NSString stringWithUTF8String:json]; free(json);
    if (!value.length) return _mathResults ?: @[];
    NSString *presentation=[NSString stringWithFormat:@"%d|%.6f|%@",_dark,_zoom,value];
    if (![_mathJSON isEqual:presentation]) {
        _mathJSON=presentation;
        _mathResults=[NSJSONSerialization JSONObjectWithData:[value dataUsingEncoding:NSUTF8StringEncoding] options:0 error:NULL];
    }
    return _mathResults ?: @[];
}
- (NSArray *)mathForFragment:(NSDictionary *)document replacingRange:(NSRange)range {
    if (range.location==NSNotFound) return [self mathForDocument:document];
    NSMutableDictionary *request=[document mutableCopy];
    request[@"before"]=[self documentForRange:NSMakeRange(0,range.location)];
    request[@"after"]=[self documentForRange:NSMakeRange(NSMaxRange(range),_text.string.length-NSMaxRange(range))];
    return [self mathForDocument:request];
}
- (NSDictionary *)nextMath {
    if (!_preparedMath || _mathIndex>=_preparedMath.count) return nil;
    return _preparedMath[_mathIndex++];
}
- (NSImage *)mathImage:(NSDictionary *)result {
    NSData *data=[[NSData alloc] initWithBase64EncodedString:result[@"image"] ?: @"" options:0];
    NSImage *image=data.length ? [[NSImage alloc] initWithData:data] : nil;
    if (image) image.size=NSMakeSize(image.size.width/2,image.size.height/2);
    return image;
}
- (BOOL)revealInlineMathAt:(NSUInteger)location {
    if (!_text.editable || _text.hasMarkedText || location>=_text.string.length) return NO;
    NSDictionary *attrs=[_text.textStorage attributesAtIndex:location effectiveRange:NULL];
    NSString *source=attrs[CRMathText];
    if (!source || !attrs[NSAttachmentAttributeName]) return NO;
    NSMutableDictionary *typing=[attrs mutableCopy];
    for (NSString *key in @[NSAttachmentAttributeName,CRMathText,CRMathExpr,CRMathNumber,CRMathRendered]) [typing removeObjectForKey:key];
    typing[CRInlineSource]=NSUUID.UUID.UUIDString;
    NSAttributedString *value=[[NSAttributedString alloc] initWithString:source attributes:typing];
    [self replace:NSMakeRange(location,1) with:value selection:NSMakeRange(location+MIN((NSUInteger)1,source.length),0)];
    _text.typingAttributes=typing;
    return YES;
}
- (void)finishInlineSource:(BOOL)force {
    if (_suppress || _text.hasMarkedText || _undo.isUndoing || _undo.isRedoing) return;
    NSMutableArray *pending=[NSMutableArray array];
    NSRange selection=_text.selectedRange;
    [_text.textStorage enumerateAttribute:CRInlineSource inRange:NSMakeRange(0,_text.textStorage.length) options:0 usingBlock:^(id source, NSRange range, BOOL *stop) {
        BOOL active=selection.length ? NSIntersectionRange(selection,range).length>0 : selection.location>=range.location && selection.location<=NSMaxRange(range);
        if (source && (force || !active)) [pending addObject:[NSValue valueWithRange:range]];
    }];
    for (NSValue *value in pending.reverseObjectEnumerator) {
        NSRange range=value.rangeValue;
        NSString *source=[_text.string substringWithRange:range];
        NSTextTableBlock *cell=CRCellAt(self,range.location);
        char *json=cr_fold_inline((char *)source.UTF8String,cell?1:0);
        if (!json) continue;
        NSDictionary *document=[NSJSONSerialization JSONObjectWithData:[[NSString stringWithUTF8String:json] dataUsingEncoding:NSUTF8StringEncoding] options:0 error:NULL]; free(json);
        NSArray *paragraphs=document[@"paragraphs"];
        if (paragraphs.count!=1 || !CRKind(paragraphs[0],@"body")) continue;
        NSArray *positions=document[@"positions"];
        if (positions.count!=source.length+1) continue;
        NSArray *prepared=[self mathForFragment:document replacingRange:range];
        BOOL failed=NO;
        for (NSDictionary *result in prepared) if (![result[@"image"] length]) failed=YES;
        if (failed) continue;
        NSArray *previous=_preparedMath; NSUInteger previousIndex=_mathIndex;
        _preparedMath=prepared; _mathIndex=0;
        NSDictionary *paragraph=CRParagraphAt(self,range.location);
        NSAttributedString *content=[self attributedRuns:paragraphs[0][@"runs"] paragraph:paragraph extra:cell ? @{CRTableCell:cell} : @{}];
        _preparedMath=previous; _mathIndex=previousIndex;
        NSRange caret=_text.selectedRange;
        NSUInteger (^mapPosition)(NSUInteger)=^NSUInteger(NSUInteger position) {
            if (position<=range.location) return position;
            if (position>=NSMaxRange(range)) return position-range.length+content.length;
            return range.location+MIN([positions[position-range.location] unsignedIntegerValue],content.length);
        };
        NSUInteger end=mapPosition(NSMaxRange(caret));
        caret.location=mapPosition(caret.location); caret.length=end-caret.location;
        [self replace:range with:content selection:caret];
        [self updateTypingParagraph];
    }
}
- (BOOL)finishSourceEditing {
    if (_suppress || !_text.editable || _text.hasMarkedText) return NO;
    NSUInteger location=_text.selectedRange.location;
    NSRange selection=_text.selectedRange;
    __block BOOL inlineSource=NO;
    [_text.textStorage enumerateAttribute:CRInlineSource inRange:NSMakeRange(0,_text.string.length) options:0 usingBlock:^(id source, NSRange range, BOOL *stop) {
        if (source && (selection.length ? NSIntersectionRange(selection,range).length>0 : location>=range.location && location<=NSMaxRange(range))) inlineSource=YES;
    }];
    if (inlineSource) {
        [self finishInlineSource:YES]; return YES;
    }
    NSRange group=[self draftGroupAt:location];
    if (group.location==NSNotFound) return NO;
    NSString *source=[[_text.string substringWithRange:CRContent(_text.string,group)] stringByTrimmingCharactersInSet:NSCharacterSet.whitespaceAndNewlineCharacterSet];
    if (![source hasPrefix:@"$$"]) return NO;
    [self renderDraftGroup:group]; return YES;
}
- (void)refreshAllMath {
    if (_suppress || _text.hasMarkedText) return;
    [self finishInlineSource:YES];
    NSMutableArray *groups=[NSMutableArray array];
    for (NSValue *value in CRRanges(_text.string)) {
        if (!CRKind(CRParagraphAt(self,value.rangeValue.location),@"source")) continue;
        NSRange group=[self draftGroupAt:value.rangeValue.location];
        if (group.location==NSNotFound) continue;
        NSValue *candidate=[NSValue valueWithRange:group];
        if (![groups containsObject:candidate]) [groups addObject:candidate];
    }
    for (NSValue *value in groups.reverseObjectEnumerator) {
        NSString *source=[[_text.string substringWithRange:CRContent(_text.string,value.rangeValue)] stringByTrimmingCharactersInSet:NSCharacterSet.whitespaceAndNewlineCharacterSet];
        if ([source hasPrefix:@"$$"]) [self renderDraftGroup:value.rangeValue];
    }
    [self refreshMath];
}
// hasMath: any inline or display formula is present.
- (BOOL)hasMath {
    __block BOOL found=NO;
    NSRange all=NSMakeRange(0,_text.string.length);
    [_text.textStorage enumerateAttribute:CRMathExpr inRange:all options:0 usingBlock:^(id value, NSRange r, BOOL *stop) { if (value) found=*stop=YES; }];
    if (!found) [_text.textStorage enumerateAttribute:CRParagraph inRange:all options:0 usingBlock:^(NSDictionary *p, NSRange r, BOOL *stop) { if (CRKind(p,@"math")) found=*stop=YES; }];
    return found;
}
- (void)refreshMath {
    if (_suppress || _text.hasMarkedText) return;
    // Numbering is document-wide, so a formula needs a full pass, but a note
    // without any must not serialize itself on every keystroke.
    if (!_mathResults.count && ![self hasMath]) return;
    NSArray *results=[self mathForDocument:[self document]];
    NSMutableArray *ranges=[NSMutableArray array];
    for (NSValue *value in CRRanges(_text.string)) {
        NSRange range=CRContent(_text.string,value.rangeValue);
        if (CRKind(CRParagraphAt(self,range.location),@"math") && [[_text.string substringWithRange:range] isEqual:@"\uFFFC"]) {
            [ranges addObject:[NSValue valueWithRange:range]];
        } else {
            [_text.textStorage enumerateAttributesInRange:range options:0 usingBlock:^(NSDictionary *attrs, NSRange run, BOOL *stop) {
                if (attrs[CRMathExpr]) [ranges addObject:[NSValue valueWithRange:run]];
            }];
        }
    }
    if (ranges.count!=results.count) return;
    BOOL previous=_suppress; _suppress=YES;
    BOOL changed=NO;
    for (NSUInteger i=0;i<ranges.count;i++) {
        NSRange range=[ranges[i] rangeValue];
        NSDictionary *result=results[i];
        if ([_text.textStorage attribute:CRMathRendered atIndex:range.location effectiveRange:NULL]==result) continue;
        changed=YES;
        NSImage *image=[self mathImage:result];
        NSDictionary *paragraph=CRParagraphAt(self,range.location);
        BOOL display=CRKind(paragraph,@"math");
        if (range.length==1 && [_text.textStorage attribute:NSAttachmentAttributeName atIndex:range.location effectiveRange:NULL]) {
            if (!image) {
                NSString *source=display ? paragraph[@"raw"] : [_text.textStorage attribute:CRMathText atIndex:range.location effectiveRange:NULL];
                NSDictionary *style=@{NSFontAttributeName:[NSFont monospacedSystemFontOfSize:14*_zoom weight:NSFontWeightRegular],NSForegroundColorAttributeName:[NSColor colorWithWhite:_dark ? 0.91 : 0.16 alpha:1]};
                NSSize size=[source sizeWithAttributes:style];
                size.width=MIN(MAX(1,size.width),1000*_zoom); size.height=MAX(1,size.height);
                image=[NSImage imageWithSize:size flipped:NO drawingHandler:^BOOL(NSRect rect) { [source drawInRect:rect withAttributes:style]; return YES; }];
            }
            // Replace the attachment attribute, never mutate a shared undo
            // snapshot's image or rebuild the editor's text and selection.
            NSTextAttachment *attachment=[NSTextAttachment new]; attachment.image=image;
            attachment.bounds=display ? [self imageBounds:image spec:nil] : NSMakeRect(0,-image.size.height*0.22,image.size.width,image.size.height);
            [_text.textStorage addAttribute:NSAttachmentAttributeName value:attachment range:range];
        }
        [_text.textStorage addAttributes:@{CRMathRendered:result,CRMathNumber:result[@"number"] ?: @""} range:range];
        if ([result[@"error"] length]) [_text.textStorage addAttribute:NSToolTipAttributeName value:result[@"error"] range:range];
        else [_text.textStorage removeAttribute:NSToolTipAttributeName range:range];
    }
    if (changed) {
        [_text.layoutManager invalidateLayoutForCharacterRange:NSMakeRange(0,_text.textStorage.length) actualCharacterRange:NULL];
        [_text setNeedsDisplay:YES];
    }
    _suppress=previous;
}

// Footnote numbering and hover help are presentation derived from live runs.
// Source tokens and definition identities stay in the Markdown model.
- (NSArray *)footnoteRuns:(NSArray *)runs {
    NSMutableArray *result=[NSMutableArray array];
    for (NSDictionary *run in runs) {
        if (![run[@"source"] boolValue]) { [result addObject:run]; continue; }
        char *json=cr_parse_inline((char *)[run[@"text"] UTF8String],0);
        if (!json) continue;
        NSDictionary *doc=[NSJSONSerialization JSONObjectWithData:[[NSString stringWithUTF8String:json] dataUsingEncoding:NSUTF8StringEncoding] options:0 error:NULL]; free(json);
        for (NSDictionary *p in doc[@"paragraphs"]) [result addObjectsFromArray:p[@"runs"] ?: @[]];
    }
    return result;
}
- (void)prepareFootnotes:(NSDictionary *)document {
    NSMutableDictionary *definitions=[NSMutableDictionary dictionary], *groups=[NSMutableDictionary dictionary];
    NSMutableArray *allRuns=[NSMutableArray array];
    for (NSDictionary *p in document[@"paragraphs"]) {
        NSMutableArray *runs=[[self footnoteRuns:p[@"runs"]] mutableCopy];
        for (NSArray *row in p[@"rows"]) for (NSDictionary *cell in row) [runs addObjectsFromArray:[self footnoteRuns:cell[@"runs"]]];
        [allRuns addObjectsFromArray:runs];
        NSString *label=p[@"footnoteId"], *group=p[@"footnoteGroup"] ?: @"";
        if (!label.length || (groups[label] && ![groups[label] isEqual:group])) continue;
        groups[label]=group;
        NSMutableString *text=[NSMutableString string];
        for (NSDictionary *run in runs) [text appendString:[run[@"break"] boolValue] ? @"\n" : (run[@"footnote"] ?: run[@"text"] ?: @"")];
        NSString *previous=definitions[label];
        definitions[label]=previous ? [previous stringByAppendingFormat:@"\n\n%@",text] : text;
    }
    NSMutableDictionary *presentation=[NSMutableDictionary dictionary];
    NSUInteger number=0;
    for (NSDictionary *run in allRuns) {
        NSString *label=run[@"footnote"];
        if (!label.length || presentation[label]) continue;
        NSString *text=definitions[label];
        presentation[label]=@{@"number":text ? [NSString stringWithFormat:@"%lu",(unsigned long)++number] : @"?",@"tooltip":text ?: [NSString stringWithFormat:@"未找到脚注定义：%@",label]};
    }
    _footnotes=presentation;
}
- (NSAttributedString *)footnote:(NSDictionary *)run attributes:(NSDictionary *)attributes {
    NSString *label=run[@"footnote"];
    NSDictionary *presentation=_footnotes[label];
    NSString *number=presentation[@"number"] ?: @"?";
    NSString *key=[NSString stringWithFormat:@"footnote|%d|%.6f|%@",_dark,_zoom,number];
    NSImage *image=_mathCache[key];
    if (!image) {
        NSDictionary *style=@{NSFontAttributeName:[NSFont systemFontOfSize:11*_zoom weight:NSFontWeightMedium],NSForegroundColorAttributeName:NSColor.linkColor};
        NSSize size=[number sizeWithAttributes:style]; size.width=ceil(size.width+3*_zoom); size.height=ceil(size.height);
        image=[NSImage imageWithSize:size flipped:NO drawingHandler:^BOOL(NSRect rect) { [number drawAtPoint:NSMakePoint(1,0) withAttributes:style]; return YES; }];
        _mathCache[key]=image;
    }
    NSTextAttachment *attachment=[NSTextAttachment new];
    attachment.image=image;
    image.accessibilityDescription=[NSString stringWithFormat:@"脚注 %@：%@",label,presentation[@"tooltip"] ?: @""];
    attachment.bounds=NSMakeRect(0,5*_zoom,image.size.width,image.size.height);
    NSMutableAttributedString *value=[[NSAttributedString attributedStringWithAttachment:attachment] mutableCopy];
    NSMutableDictionary *attrs=[attributes mutableCopy];
    attrs[NSAttachmentAttributeName]=attachment; attrs[CRFootnote]=run; attrs[CRFootnoteNumber]=number;
    attrs[NSToolTipAttributeName]=presentation[@"tooltip"] ?: [NSString stringWithFormat:@"脚注：%@",label];
    [value addAttributes:attrs range:NSMakeRange(0,value.length)];
    return value;
}
- (void)refreshFootnotes {
    if (_text.hasMarkedText) return;
    [self prepareFootnotes:[self document]];
    NSAttributedString *copy=[_text.textStorage copy];
    BOOL wasSuppress=_suppress; _suppress=YES;
    [copy enumerateAttributesInRange:NSMakeRange(0,copy.length) options:0 usingBlock:^(NSDictionary *attrs, NSRange range, BOOL *stop) {
        id run=attrs[CRFootnote];
        if (![run isKindOfClass:NSDictionary.class]) return;
        NSDictionary *updated=[[self footnote:run attributes:attrs] attributesAtIndex:0 effectiveRange:NULL];
        [self.text.textStorage addAttributes:updated range:range];
    }];
    _suppress=wasSuppress;
}
- (void)navigateFootnote:(BOOL)definition atLocation:(NSUInteger)location {
    if (_text.hasMarkedText || location>_text.string.length) return;
    NSString *label=[self footnoteLabelAt:location];
    if (!label.length) return;
    [self finishInlineSource:YES];
    __block NSUInteger target=NSNotFound;
    if (definition) {
        for (NSValue *value in CRRanges(_text.string)) {
            NSDictionary *p=CRParagraphAt(self,value.rangeValue.location);
            if ([p[@"footnoteId"] isEqual:label]) { target=value.rangeValue.location; break; }
        }
    } else [_text.textStorage enumerateAttribute:CRFootnote inRange:NSMakeRange(0,_text.string.length) options:0 usingBlock:^(id run, NSRange range, BOOL *stop) {
        if ([run[@"footnote"] isEqual:label]) { target=range.location; *stop=YES; }
    }];
    if (target==NSNotFound) return;
    [_text.window makeFirstResponder:_text]; _text.selectedRange=NSMakeRange(target,0);
    [_text scrollRangeToVisible:_text.selectedRange];
}
- (NSString *)footnoteLabelAt:(NSUInteger)location {
    if (location>_text.string.length) return nil;
    NSString *label=nil;
    if (location<_text.string.length) label=[_text.textStorage attribute:CRFootnote atIndex:location effectiveRange:NULL][@"footnote"];
    NSUInteger hit=location;
    if (hit==_text.string.length && hit) hit--;
    if (!label && hit<_text.string.length && [_text.textStorage attribute:CRInlineSource atIndex:hit effectiveRange:NULL]) {
        NSRange expanded;
        [_text.textStorage attribute:CRInlineSource atIndex:hit longestEffectiveRange:&expanded inRange:NSMakeRange(0,_text.string.length)];
        NSString *source=[_text.string substringWithRange:expanded];
        char *json=cr_fold_inline((char *)source.UTF8String,CRCellAt(self,hit) ? 1 : 0);
        NSDictionary *doc=json ? [NSJSONSerialization JSONObjectWithData:[[NSString stringWithUTF8String:json] dataUsingEncoding:NSUTF8StringEncoding] options:0 error:NULL] : nil;
        if (json) free(json);
        NSArray *positions=doc[@"positions"];
        NSUInteger offset=MIN(location-expanded.location,source.length);
        NSUInteger caret=offset<positions.count ? [positions[offset] unsignedIntegerValue] : NSNotFound, start=0;
        for (NSDictionary *p in doc[@"paragraphs"]) for (NSDictionary *run in p[@"runs"]) {
            NSUInteger length=[run[@"math"] length] || [run[@"footnote"] length] || [run[@"break"] boolValue] ? 1 : [run[@"text"] length];
            if ([run[@"footnote"] length] && caret>=start && caret<=start+length) { label=run[@"footnote"]; break; }
            start+=length;
        }
    }
    return label ?: CRParagraphAt(self,location)[@"footnoteId"];
}
- (NSArray *)footnoteFragment:(NSArray *)paragraphs container:(NSDictionary *)container {
    if (![container[@"footnoteId"] length]) return paragraphs;
    NSMutableArray *result=[NSMutableArray array];
    for (NSUInteger i=0;i<paragraphs.count;) {
        NSDictionary *p=paragraphs[i];
        if (![p[@"footnoteId"] length] || [p[@"footnoteGroup"] isEqual:container[@"footnoteGroup"]]) { [result addObject:CRInFootnote(p,container)]; i++; continue; }
        NSUInteger end=i+1;
        while (end<paragraphs.count && [paragraphs[end][@"footnoteGroup"] isEqual:p[@"footnoteGroup"]]) end++;
        NSData *data=[NSJSONSerialization dataWithJSONObject:@{@"paragraphs":[paragraphs subarrayWithRange:NSMakeRange(i,end-i)]} options:0 error:NULL];
        char *json=data ? cr_markdown_json((char *)[[[NSString alloc] initWithData:data encoding:NSUTF8StringEncoding] UTF8String]) : NULL;
        NSString *source=json ? [NSString stringWithUTF8String:json] : @"";
        if (json) free(json);
        source=[source stringByTrimmingCharactersInSet:NSCharacterSet.newlineCharacterSet];
        NSArray *lines=[source componentsSeparatedByString:@"\n"];
        for (NSUInteger line=0;line<lines.count;line++) {
            NSString *text=lines[line]; if ([text hasSuffix:@"\r"]) text=[text substringToIndex:text.length-1];
            NSString *separator=line+1<lines.count ? @"\n" : (paragraphs[end-1][@"separator"] ?: @"");
            [result addObject:CRInFootnote(@{@"kind":@"source",@"runs":@[@{@"text":text}],@"raw":text,@"separator":separator},container)];
        }
        i=end;
    }
    return result;
}
- (BOOL)allowFootnoteRange:(NSRange)selection {
    if (!selection.length) return YES;
    NSMutableArray *groups=[NSMutableArray array];
    NSString *previous=nil;
    for (NSValue *value in CRRanges(_text.string)) {
        NSRange range=value.rangeValue;
        NSString *group=CRParagraphAt(self,range.location)[@"footnoteGroup"];
        if (group.length) {
            if (![previous isEqual:group]) [groups addObject:[NSMutableDictionary dictionaryWithDictionary:@{@"range":value,@"end":@(NSMaxRange(CRContent(_text.string,range)))}]];
            else {
                NSMutableDictionary *entry=groups.lastObject;
                entry[@"range"]=[NSValue valueWithRange:NSUnionRange([entry[@"range"] rangeValue],range)];
                entry[@"end"]=@(NSMaxRange(CRContent(_text.string,range)));
            }
        }
        previous=group;
    }
    for (NSDictionary *group in groups) {
        NSRange range=[group[@"range"] rangeValue];
        BOOL touches=NSIntersectionRange(selection,range).length || NSMaxRange(selection)==range.location;
        BOOL whole=selection.location<=range.location && NSMaxRange(selection)>=NSMaxRange(range);
        if (touches && !whole && (selection.location<range.location || NSMaxRange(selection)>[group[@"end"] unsignedIntegerValue])) return NO;
    }
    return YES;
}
- (void)insertFootnote {
    if (!_text.editable || _text.hasMarkedText) return;
    [self finishInlineSource:YES];
    NSRange selection=_text.selectedRange;
    if (![self allowRange:selection]) return;
    NSDictionary *paragraph=CRParagraphAt(self,selection.location);
    if ([@[@"code",@"source",@"opaque",@"math",@"image",@"diagram",@"toc"] containsObject:paragraph[@"kind"]]) return;
    NSTextTableBlock *cell=CRCellAt(self,selection.location);
    if (cell && selection.length && [[_text.string substringWithRange:selection] containsString:@"\n"]) return;
    NSMutableSet *used=[NSMutableSet set];
    NSDictionary *document=[self document];
    for (NSDictionary *p in document[@"paragraphs"]) {
        if (p[@"footnoteId"]) [used addObject:p[@"footnoteId"]];
        NSMutableArray *runs=[[self footnoteRuns:p[@"runs"]] mutableCopy];
        for (NSArray *row in p[@"rows"]) for (NSDictionary *entry in row) [runs addObjectsFromArray:[self footnoteRuns:entry[@"runs"]]];
        for (NSDictionary *run in runs) if (run[@"footnote"]) [used addObject:run[@"footnote"]];
    }
    NSUInteger number=1;
    while ([used containsObject:[NSString stringWithFormat:@"%lu",(unsigned long)number]]) number++;
    NSString *label=[NSString stringWithFormat:@"%lu",(unsigned long)number];
    NSDictionary *run=@{@"text":[NSString stringWithFormat:@"[^%@]",label],@"footnote":label};
    NSMutableDictionary *attrs=[@{CRParagraph:paragraph} mutableCopy];
    if (cell) attrs[CRTableCell]=cell;
    NSAttributedString *reference=[self footnote:run attributes:[self visual:attrs paragraph:paragraph]];
    NSMutableAttributedString *after=[_text.textStorage mutableCopy];
    [after replaceCharactersInRange:selection withAttributedString:reference];
    NSDictionary *last=CRParagraphAt(self,_text.string.length);
    if (after.length) [after appendAttributedString:[[NSAttributedString alloc] initWithString:@"\n" attributes:[self visual:@{CRParagraph:last} paragraph:last]]];
    NSDictionary *definition=@{@"kind":@"body",@"runs":@[],@"footnoteId":label,@"footnoteGroup":NSUUID.UUID.UUIDString,@"footnoteMarker":[NSString stringWithFormat:@"[^%@]: ",label],@"footnoteIndent":@"    "};
    NSDictionary *typing=[self visual:@{CRParagraph:definition} paragraph:definition];
    [self beginStructuralCommand]; [_undo beginUndoGrouping];
    [self restore:after selection:NSMakeRange(after.length,0) typing:typing ending:definition];
    [_undo endUndoGrouping];
}

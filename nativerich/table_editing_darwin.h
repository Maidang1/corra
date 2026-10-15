// Included in CREditor's implementation; all edits use the live text storage.
- (void)tableCommand:(NSString *)command {
    if (!_text.editable || _text.hasMarkedText) return;
    NSTextTableBlock *cell = CRCellAt(self,_text.selectedRange.location);
    if (!cell) return;
    NSRange tableRange = CRTableRange(self,cell.table);
    NSRange active = [_text.string paragraphRangeForRange:NSMakeRange(_text.selectedRange.location,0)];
    if (NSMaxRange(_text.selectedRange) > NSMaxRange(active)) return;
    NSMutableDictionary *p = [[self documentForRange:tableRange][@"paragraphs"][0] mutableCopy];
    NSMutableArray *rows = [NSMutableArray array];
    for (NSArray *row in p[@"rows"]) [rows addObject:[row mutableCopy]];
    NSInteger r = cell.startingRow, c = cell.startingColumn;
    NSInteger columns = cell.table.numberOfColumns;
    NSMutableArray *align = [p[@"alignments"] mutableCopy] ?: [NSMutableArray array];
    while (align.count < columns) [align addObject:@"none"];
    if ([command isEqual:@"table:row-above"] || [command isEqual:@"table:row-below"]) {
        // The header stays the first row; insertion above it creates a body row.
        NSInteger target = MAX(1,r + ([command hasSuffix:@"below"] ? 1 : 0));
        NSMutableArray *row = [NSMutableArray array];
        for (NSInteger i=0; i<columns; i++) [row addObject:@{@"runs":@[]}];
        [rows insertObject:row atIndex:target]; r=target;
    } else if ([command isEqual:@"table:delete-row"]) {
        if (r==0 || rows.count<=2) return;
        [rows removeObjectAtIndex:r]; r=MIN(r,(NSInteger)rows.count-1);
    } else if ([command isEqual:@"table:column-left"] || [command isEqual:@"table:column-right"]) {
        NSInteger target = c + ([command hasSuffix:@"right"] ? 1 : 0);
        for (NSMutableArray *row in rows) [row insertObject:@{@"runs":@[]} atIndex:target];
        [align insertObject:@"none" atIndex:target]; c=target; columns++;
    } else if ([command isEqual:@"table:delete-column"]) {
        if (columns<=1) return;
        for (NSMutableArray *row in rows) [row removeObjectAtIndex:c];
        [align removeObjectAtIndex:c]; columns--; c=MIN(c,columns-1);
    } else if ([command isEqual:@"table:row-up"] || [command isEqual:@"table:row-down"]) {
        NSInteger target = r + ([command hasSuffix:@"up"] ? -1 : 1);
        if (r==0 || target<1 || target>=rows.count) return;
        [rows exchangeObjectAtIndex:r withObjectAtIndex:target]; r=target;
    } else if ([command isEqual:@"table:column-leftward"] || [command isEqual:@"table:column-rightward"]) {
        NSInteger target = c + ([command hasSuffix:@"leftward"] ? -1 : 1);
        if (target<0 || target>=columns) return;
        for (NSMutableArray *row in rows) [row exchangeObjectAtIndex:c withObjectAtIndex:target];
        [align exchangeObjectAtIndex:c withObjectAtIndex:target]; c=target;
    } else if ([command hasPrefix:@"table:align-"]) {
        NSString *alignment = [command substringFromIndex:12];
        if (![@[@"left",@"center",@"right",@"none"] containsObject:alignment] || [align[c] isEqual:alignment]) return;
        align[c]=alignment;
    } else return;
    p[@"rows"]=rows; p[@"alignments"]=align;
    [self replaceTable:tableRange paragraph:p row:r column:c];
}
- (void)replaceTable:(NSRange)tableRange paragraph:(NSDictionary *)p row:(NSInteger)r column:(NSInteger)c {
    [self beginStructuralCommand];
    NSDictionary *ending = _emptyParagraph;
    NSAttributedString *value = [self attributedDocument:@{@"paragraphs":@[p]} replacingRange:tableRange];
    _emptyParagraph=ending;
    // Measure actual attributed cells: formulas may fall back to literal source.
    __block NSUInteger target=0;
    [value enumerateAttribute:CRTableCell inRange:NSMakeRange(0,value.length) options:0 usingBlock:^(NSTextTableBlock *candidate, NSRange range, BOOL *stop) {
        if (candidate && candidate.startingRow==r && candidate.startingColumn==c) { target=range.location; *stop=YES; }
    }];
    [self replace:tableRange with:value selection:NSMakeRange(tableRange.location+target,0)];
    [self updateTypingParagraph];
    [_text scrollRangeToVisible:_text.selectedRange];
}
- (BOOL)pasteTableText:(NSString *)text {
    if ([text rangeOfString:@"\t"].location==NSNotFound) return NO;
    if (!_text.editable || _text.hasMarkedText) return YES;
    NSTextTableBlock *cell=CRCellAt(self,_text.selectedRange.location);
    if (!cell) return NO;
    NSRange active=[_text.string paragraphRangeForRange:NSMakeRange(_text.selectedRange.location,0)];
    if (NSMaxRange(_text.selectedRange)>NSMaxRange(active)) return YES;
    NSRange range=CRTableRange(self,cell.table);
    NSMutableDictionary *p=[[self documentForRange:range][@"paragraphs"][0] mutableCopy];
    NSMutableArray *rows=[NSMutableArray array];
    for (NSArray *row in p[@"rows"]) [rows addObject:[row mutableCopy]];
    NSString *normalized=[[text stringByReplacingOccurrencesOfString:@"\r\n" withString:@"\n"] stringByReplacingOccurrencesOfString:@"\r" withString:@"\n"];
    NSMutableArray *lines=[[normalized componentsSeparatedByString:@"\n"] mutableCopy];
    if (lines.count>1 && [lines.lastObject length]==0) [lines removeLastObject];
    NSInteger columns=cell.table.numberOfColumns;
    for (NSString *line in lines) columns=MAX(columns,cell.startingColumn+[line componentsSeparatedByString:@"\t"].count);
    NSMutableArray *align=[p[@"alignments"] mutableCopy] ?: [NSMutableArray array];
    while (align.count<columns) [align addObject:@"none"];
    while (rows.count<MAX(2,cell.startingRow+lines.count)) [rows addObject:[NSMutableArray array]];
    for (NSMutableArray *row in rows) while (row.count<columns) [row addObject:@{@"runs":@[]}];
    for (NSUInteger ri=0; ri<lines.count; ri++) {
        NSArray *values=[lines[ri] componentsSeparatedByString:@"\t"];
        for (NSUInteger ci=0; ci<values.count; ci++) {
            NSString *value=values[ci];
            rows[cell.startingRow+ri][cell.startingColumn+ci]=@{@"runs":value.length ? @[@{@"text":value}] : @[]};
        }
    }
    p[@"rows"]=rows; p[@"alignments"]=align;
    [self replaceTable:range paragraph:p row:cell.startingRow column:cell.startingColumn];
    return YES;
}

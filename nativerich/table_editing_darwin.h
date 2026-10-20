// Included in CREditor's implementation; all edits use the live text storage.
- (void)tableCommand:(NSString *)command {
    if (!_text.editable || _text.hasMarkedText) return;
    NSTextTableBlock *cell = CRCellAt(self,_text.selectedRange.location);
    if (!cell) return;
    NSRange tableRange = CRTableRangeNear(self,cell.table,_text.selectedRange.location);
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
// tableMoveRow:toIndex: drags a body row (the header is pinned at 0) into the
// insertion slot target (1...rowCount), as one undo group. A slot on the
// row's own edges is a no-op, as is the header or a missing row.
- (BOOL)tableMoveRow:(NSInteger)row toIndex:(NSInteger)target inTable:(NSTextTable *)table {
    if (!_text.editable || _text.hasMarkedText || !table) return NO;
    NSRange tableRange = CRTableRange(self, table);
    if (tableRange.location == NSNotFound) return NO;
    NSMutableDictionary *p = [[self documentForRange:tableRange][@"paragraphs"][0] mutableCopy];
    NSMutableArray *rows = [NSMutableArray array];
    for (NSArray *r in p[@"rows"]) [rows addObject:[r mutableCopy]];
    if (row <= 0 || row >= (NSInteger)rows.count) return NO;
    target = MAX(1, MIN(target, (NSInteger)rows.count));
    if (target == row || target == row + 1) return NO;
    NSMutableArray *moved = rows[row];
    [rows removeObjectAtIndex:row];
    if (target > row) target--;
    [rows insertObject:moved atIndex:target];
    p[@"rows"] = rows;
    [self replaceTable:tableRange paragraph:p row:target column:0];
    return YES;
}
// tableMoveColumn:toIndex: drags a column into the insertion slot target
// (0...columnCount), carrying its alignment along. A slot on the column's own
// edges is a no-op.
- (BOOL)tableMoveColumn:(NSInteger)column toIndex:(NSInteger)target inTable:(NSTextTable *)table {
    if (!_text.editable || _text.hasMarkedText || !table) return NO;
    NSRange tableRange = CRTableRange(self, table);
    if (tableRange.location == NSNotFound) return NO;
    NSMutableDictionary *p = [[self documentForRange:tableRange][@"paragraphs"][0] mutableCopy];
    NSMutableArray *rows = [NSMutableArray array];
    for (NSArray *r in p[@"rows"]) [rows addObject:[r mutableCopy]];
    NSInteger columns = table.numberOfColumns;
    if (column < 0 || column >= columns) return NO;
    target = MAX(0, MIN(target, columns));
    if (target == column || target == column + 1) return NO;
    NSMutableArray *align = [p[@"alignments"] mutableCopy] ?: [NSMutableArray array];
    while (align.count < columns) [align addObject:@"none"];
    NSInteger final = target > column ? target - 1 : target;
    for (NSMutableArray *r in rows) {
        while (r.count < columns) [r addObject:@{@"runs":@[]}];
        id moved = r[column];
        [r removeObjectAtIndex:column];
        [r insertObject:moved atIndex:final];
    }
    id movedAlign = align[column];
    [align removeObjectAtIndex:column];
    [align insertObject:movedAlign atIndex:final];
    p[@"rows"] = rows;
    p[@"alignments"] = align;
    [self replaceTable:tableRange paragraph:p row:0 column:final];
    return YES;
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
    NSRange range=CRTableRangeNear(self,cell.table,_text.selectedRange.location);
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
// replaceAcrossTable handles a selection with exactly one end inside a table
// (Typora): the selected cells lose their text but keep the grid, the
// selected text outside is deleted, and typed text lands where the selection
// started. Without this, text after the table would merge into a cell.
- (BOOL)replaceAcrossTable:(NSRange)range withText:(NSString *)text {
    if (!_text.editable || _text.hasMarkedText || !range.length || NSMaxRange(range)>_text.string.length) return NO;
    if ([text rangeOfCharacterFromSet:NSCharacterSet.newlineCharacterSet].location!=NSNotFound || ![self allowFootnoteRange:range]) return NO;
    NSUInteger last=NSMaxRange(range)-1;
    NSTextTableBlock *head=CRCellAt(self,range.location), *tail=CRCellAt(self,last);
    if ((head!=nil)==(tail!=nil)) return NO; // inside one table, or none
    NSTextTableBlock *cell=head ?: tail;
    NSRange table=CRTableRangeNear(self,cell.table,head ? range.location : last);
    NSMutableAttributedString *after=[_text.textStorage mutableCopy];
    NSRange outer, inner;
    if (head) {
        outer=NSMakeRange(NSMaxRange(table),NSMaxRange(range)-NSMaxRange(table));
        inner=NSMakeRange(range.location,NSMaxRange(table)-range.location);
    } else {
        // Keep the line break before the table, or the line above would join
        // the first cell.
        NSUInteger end=table.location;
        if (end>range.location && [_text.string characterAtIndex:end-1]=='\n') end--;
        outer=NSMakeRange(range.location,end-range.location);
        inner=NSMakeRange(table.location,NSMaxRange(range)-table.location);
    }
    // Edit the later range first so the earlier one's offsets stay valid.
    if (head) [after deleteCharactersInRange:outer];
    // Clear each selected cell's text, last first, keeping its line break.
    for (NSValue *v in CRRangesIn(_text.string,inner).reverseObjectEnumerator) {
        NSRange cleared=NSIntersectionRange(CRContent(_text.string,v.rangeValue),inner);
        if (cleared.length) [after deleteCharactersInRange:cleared];
    }
    if (!head) [after deleteCharactersInRange:outer];
    NSDictionary *typing=range.location<_text.string.length ? [_text.textStorage attributesAtIndex:range.location effectiveRange:NULL] : _text.typingAttributes;
    if (text.length) [after insertAttributedString:[[NSAttributedString alloc] initWithString:text attributes:CRTextAttrs(typing)] atIndex:range.location];
    [self replace:NSMakeRange(0,_text.string.length) with:after selection:NSMakeRange(range.location+text.length,0)];
    return YES;
}

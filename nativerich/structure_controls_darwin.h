// Included in CRTextView's implementation so context actions and keys share
// the same transaction paths as toolbar commands.
- (BOOL)handleStructureKey:(NSEvent *)event {
    if (!self.editable || self.hasMarkedText) return NO;
    NSEventModifierFlags modifiers=event.modifierFlags & NSEventModifierFlagDeviceIndependentFlagsMask;
    modifiers &= ~(NSEventModifierFlagCapsLock | NSEventModifierFlagNumericPad | NSEventModifierFlagFunction);
    NSString *key=event.charactersIgnoringModifiers;
    BOOL table=CRCellAt(self.owner,self.selectedRange.location)!=nil;
    if (modifiers==NSEventModifierFlagOption && [key isEqual:@"\r"] && CRList(CRParagraphAt(self.owner,self.selectedRange.location))) {
        [self.owner continueListParagraph]; return YES;
    }
    if (table && modifiers==NSEventModifierFlagCommand && [key isEqual:@"\r"]) {
        [self.owner tableCommand:@"table:row-below"]; return YES;
    }
    if (modifiers==NSEventModifierFlagCommand && [key isEqual:@"\r"] && [self.owner finishMathEditing]) return YES;
    if (table && modifiers==(NSEventModifierFlagCommand|NSEventModifierFlagShift) && [key isEqual:@"\x7f"]) {
        [self.owner tableCommand:@"table:delete-row"]; return YES;
    }
    NSDictionary *paragraph=CRParagraphAt(self.owner,self.selectedRange.location);
    if (modifiers==NSEventModifierFlagCommand && ([@[@"[",@"]"] containsObject:key]) && (CRList(paragraph) || CRKind(paragraph,@"code") || CRKind(paragraph,@"quote"))) {
        [self.owner format:[key isEqual:@"["] ? @"outdent" : @"indent"]; return YES;
    }
    return NO;
}
- (BOOL)performKeyEquivalent:(NSEvent *)event {
    if (self.window.firstResponder==self && [self handleStructureKey:event]) return YES;
    return [super performKeyEquivalent:event];
}
- (void)keyDown:(NSEvent *)event {
    if (![self handleStructureKey:event]) [super keyDown:event];
}
- (void)structuralAction:(NSMenuItem *)item { [self.owner format:item.representedObject]; }
- (NSMenu *)menuForEvent:(NSEvent *)event {
    NSMenu *menu=[[super menuForEvent:event] copy] ?: [NSMenu new];
    NSPoint point=[self convertPoint:event.locationInWindow fromView:nil];
    NSUInteger location=[self characterIndexForInsertionAtPoint:point];
    if (!NSLocationInRange(location,self.selectedRange)) self.selectedRange=NSMakeRange(location,0);
    BOOL table=CRCellAt(self.owner,location)!=nil;
    BOOL list=CRList(CRParagraphAt(self.owner,location));
    if (!table && !list) return menu;
    [menu addItem:NSMenuItem.separatorItem];
    NSArray *entries=table ? @[
        @[@"在上方插入行",@"table:row-above"], @[@"在下方插入行",@"table:row-below"],
        @[@"删除当前行",@"table:delete-row"], @[@"上移当前行",@"table:row-up"], @[@"下移当前行",@"table:row-down"],
        @[@"在左侧插入列",@"table:column-left"], @[@"在右侧插入列",@"table:column-right"],
        @[@"删除当前列",@"table:delete-column"], @[@"左移当前列",@"table:column-leftward"], @[@"右移当前列",@"table:column-rightward"],
        @[@"当前列左对齐",@"table:align-left"], @[@"当前列居中",@"table:align-center"], @[@"当前列右对齐",@"table:align-right"]
    ] : @[@[@"增加列表缩进",@"indent"], @[@"减少列表缩进",@"outdent"], @[@"在列表项内新建段落",@"list-paragraph"]];
    for (NSArray *entry in entries) {
        NSMenuItem *item=[[NSMenuItem alloc] initWithTitle:entry[0] action:@selector(structuralAction:) keyEquivalent:@""];
        item.target=self; item.representedObject=entry[1];
        [menu addItem:item];
    }
    return menu;
}
- (BOOL)validateMenuItem:(NSMenuItem *)item {
    if (item.action==@selector(structuralAction:)) return self.editable && !self.hasMarkedText;
    return [super validateMenuItem:item];
}

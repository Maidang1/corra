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
    if (modifiers==NSEventModifierFlagCommand && [key isEqual:@"\r"] && [self.owner finishSourceEditing]) return YES;
    if (table && modifiers==NSEventModifierFlagCommand && [key isEqual:@"\r"]) {
        [self.owner tableCommand:@"table:row-below"]; return YES;
    }
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
- (void)moveRight:(id)sender {
    NSRange previous=self.selectedRange;
    BOOL source=previous.location<self.string.length && [self.textStorage attribute:CRInlineSource atIndex:previous.location effectiveRange:NULL]!=nil;
    if (!source && previous.location) source=[self.textStorage attribute:CRInlineSource atIndex:previous.location-1 effectiveRange:NULL]!=nil;
    NSUInteger oldLength=self.string.length;
    [super moveRight:sender];
    if (!self.selectedRange.length && !source && ![self.owner revealInlineAt:self.selectedRange.location] && self.string.length==oldLength && previous.location<self.string.length && self.selectedRange.location>previous.location) [self.owner revealInlineAt:previous.location];
}
- (void)moveLeft:(id)sender { [super moveLeft:sender]; if (!self.selectedRange.length) [self.owner revealInlineAt:self.selectedRange.location]; }
- (void)moveUp:(id)sender { [super moveUp:sender]; if (!self.selectedRange.length) [self.owner revealInlineAt:self.selectedRange.location]; }
- (void)moveDown:(id)sender { [super moveDown:sender]; if (!self.selectedRange.length) [self.owner revealInlineAt:self.selectedRange.location]; }
- (void)structuralAction:(NSMenuItem *)item { [self.owner format:item.representedObject]; }
- (void)codeLanguageAction:(NSMenuItem *)item {
    NSDictionary *target=item.representedObject;
    if (![target isKindOfClass:NSDictionary.class] || [target[@"revision"] unsignedIntegerValue]!=self.owner.documentRevision) return;
    NSString *command=target[@"command"];
    if ([command hasPrefix:@"code-language:"]) [self.owner setCodeLanguage:[command substringFromIndex:14] atLocation:[target[@"location"] unsignedIntegerValue]];
}
- (void)chooseCodeLanguage:(NSMenuItem *)sender {
    NSDictionary *target=sender.representedObject;
    NSUInteger location=[target[@"location"] unsignedIntegerValue];
    if (!self.editable || self.hasMarkedText || location>self.string.length || [target[@"revision"] unsignedIntegerValue]!=self.owner.documentRevision || !CRKind(CRParagraphAt(self.owner,location),@"code")) return;
    NSAlert *alert=[NSAlert new];
    alert.messageText=@"代码语言";
    alert.informativeText=@"输入语言名称，例如 go、python、rust；纯文本使用 text。";
    [alert addButtonWithTitle:@"应用"]; [alert addButtonWithTitle:@"取消"];
    NSTextField *field=[[NSTextField alloc] initWithFrame:NSMakeRect(0,0,280,24)];
    field.accessibilityLabel=@"代码语言";
    for (NSDictionary *block in [self.owner codeBlocks]) {
        NSRange range=[block[@"range"] rangeValue];
        if (NSLocationInRange(location,range) || (range.length==0 && location==range.location)) field.stringValue=block[@"language"];
    }
    alert.accessoryView=field; alert.window.initialFirstResponder=field;
    NSString *snapshot=[self.string copy];
    NSUInteger revision=self.owner.documentRevision;
    [alert beginSheetModalForWindow:self.window completionHandler:^(NSModalResponse response) {
        NSString *language=[field.stringValue stringByTrimmingCharactersInSet:NSCharacterSet.whitespaceAndNewlineCharacterSet];
        if (!language.length) language=@"text";
        if (response==NSAlertFirstButtonReturn && self.window && self.owner.documentRevision==revision && [self.string isEqual:snapshot] && !self.hasMarkedText) [self.owner setCodeLanguage:language atLocation:location];
    }];
}
- (NSMenu *)menuForEvent:(NSEvent *)event {
    NSMenu *menu=[[super menuForEvent:event] copy] ?: [NSMenu new];
    NSPoint point=[self convertPoint:event.locationInWindow fromView:nil];
    NSUInteger location=[self characterIndexForInsertionAtPoint:point];
    if (!NSLocationInRange(location,self.selectedRange)) self.selectedRange=NSMakeRange(location,0);
    BOOL table=CRCellAt(self.owner,location)!=nil;
    BOOL list=CRList(CRParagraphAt(self.owner,location));
    if (CRKind(CRParagraphAt(self.owner,location),@"code")) {
        [menu addItem:NSMenuItem.separatorItem];
        NSMenuItem *language=[[NSMenuItem alloc] initWithTitle:@"代码语言…" action:@selector(chooseCodeLanguage:) keyEquivalent:@""];
        language.target=self; language.representedObject=@{@"location":@(location),@"revision":@(self.owner.documentRevision)}; [menu addItem:language];
        for (NSArray *entry in @[@[@"纯文本",@"text"],@[@"Go",@"go"],@[@"Python",@"python"],@[@"JavaScript",@"javascript"],@[@"TypeScript",@"typescript"],@[@"Swift",@"swift"],@[@"Rust",@"rust"],@[@"JSON",@"json"],@[@"Shell",@"bash"]]) {
            NSMenuItem *item=[[NSMenuItem alloc] initWithTitle:entry[0] action:@selector(codeLanguageAction:) keyEquivalent:@""];
            item.target=self; item.representedObject=@{@"command":[@"code-language:" stringByAppendingString:entry[1]],@"location":@(location),@"revision":@(self.owner.documentRevision)};
            [menu addItem:item];
        }
        return menu;
    }
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
    if (item.action==@selector(structuralAction:) || item.action==@selector(chooseCodeLanguage:) || item.action==@selector(codeLanguageAction:)) return self.editable && !self.hasMarkedText;
    return [super validateMenuItem:item];
}

// Included in CRTextView's implementation so context actions and keys share
// the same transaction paths as toolbar commands.
- (BOOL)handleStructureKey:(NSEvent *)event {
    if (self.hasMarkedText) return NO;
    NSEventModifierFlags modifiers=event.modifierFlags & NSEventModifierFlagDeviceIndependentFlagsMask;
    modifiers &= ~(NSEventModifierFlagCapsLock | NSEventModifierFlagNumericPad | NSEventModifierFlagFunction);
    NSString *key=event.charactersIgnoringModifiers;
    if (modifiers==(NSEventModifierFlagCommand|NSEventModifierFlagOption) && ([@[@"\uf700",@"\uf701"] containsObject:key]) && [self.owner footnoteLabelAt:self.selectedRange.location]) {
        [self.owner navigateFootnote:[key isEqual:@"\uf701"] atLocation:self.selectedRange.location]; return YES;
    }
    if (!self.editable) return NO;
    BOOL table=CRCellAt(self.owner,self.selectedRange.location)!=nil;
    if (modifiers==NSEventModifierFlagOption && [key isEqual:@"\r"] && CRList(CRParagraphAt(self.owner,self.selectedRange.location))) {
        [self.owner continueListParagraph]; return YES;
    }
    if (modifiers==NSEventModifierFlagCommand && [key isEqual:@"\r"] && [self.owner finishSourceEditing]) return YES;
    if (modifiers==NSEventModifierFlagCommand && [key isEqual:@"\r"] && [self.owner leaveCodeBlock]) return YES;
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
- (void)moveUp:(id)sender { if ([self.owner escapeBlock:NO]) return; [super moveUp:sender]; if (!self.selectedRange.length) [self.owner revealInlineAt:self.selectedRange.location]; }
- (void)moveDown:(id)sender { if ([self.owner escapeBlock:YES]) return; [super moveDown:sender]; if (!self.selectedRange.length) [self.owner revealInlineAt:self.selectedRange.location]; }
- (void)structuralAction:(NSMenuItem *)item { [self.owner format:item.representedObject]; }
- (void)footnoteAction:(NSMenuItem *)item {
    NSDictionary *target=item.representedObject;
    NSUInteger location=[target[@"location"] unsignedIntegerValue];
    if (self.hasMarkedText || [target[@"revision"] unsignedIntegerValue]!=self.owner.documentRevision || ![[self.owner footnoteLabelAt:location] isEqual:target[@"label"]]) return;
    [self.owner navigateFootnote:[target[@"command"] isEqual:@"footnote:definition"] atLocation:location];
}
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
- (void)chooseImageSize:(NSMenuItem *)sender {
    NSDictionary *target=sender.representedObject;
    NSUInteger location=[target[@"location"] unsignedIntegerValue];
    if (!self.editable || self.hasMarkedText || location>self.string.length || [target[@"revision"] unsignedIntegerValue]!=self.owner.documentRevision) return;
    if (![self.owner imageFigureRanges].count) return;
    NSAlert *alert=[NSAlert new];
    alert.messageText=@"图片大小";
    alert.informativeText=@"按像素输入宽高，留空表示自动；也可用百分比，例如 50%。";
    [alert addButtonWithTitle:@"应用"]; [alert addButtonWithTitle:@"取消"];
    NSTextField *width=[[NSTextField alloc] initWithFrame:NSMakeRect(0,32,130,24)];
    NSTextField *height=[[NSTextField alloc] initWithFrame:NSMakeRect(0,0,130,24)];
    width.placeholderString=@"宽度（像素）"; height.placeholderString=@"高度（像素）";
    NSView *box=[[NSView alloc] initWithFrame:NSMakeRect(0,0,130,60)];
    [box addSubview:width]; [box addSubview:height];
    alert.accessoryView=box; alert.window.initialFirstResponder=width;
    NSString *snapshot=[self.string copy];
    NSUInteger revision=self.owner.documentRevision;
    [alert beginSheetModalForWindow:self.window completionHandler:^(NSModalResponse response) {
        if (response!=NSAlertFirstButtonReturn || !self.window || self.owner.documentRevision!=revision || ![self.string isEqual:snapshot] || self.hasMarkedText) return;
        NSString *w=[width.stringValue stringByTrimmingCharactersInSet:NSCharacterSet.whitespaceAndNewlineCharacterSet];
        NSString *h=[height.stringValue stringByTrimmingCharactersInSet:NSCharacterSet.whitespaceAndNewlineCharacterSet];
        NSString *spec=@"";
        if (w.length && h.length) spec=[NSString stringWithFormat:@"%@x%@",w,h];
        else if (w.length) spec=[w stringByAppendingString:@"x"];
        else if (h.length) spec=[@"x" stringByAppendingString:h];
        [self.owner setImageSize:spec atLocation:location];
    }];
}
- (void)revealTOCMarker:(NSMenuItem *)item {
    NSDictionary *target=item.representedObject;
    if (self.hasMarkedText || ![target isKindOfClass:NSDictionary.class]) return;
    if ([target[@"revision"] unsignedIntegerValue]!=self.owner.documentRevision) return;
    NSUInteger location=[target[@"location"] unsignedIntegerValue];
    if (!CRKind(CRParagraphAt(self.owner,location),@"toc")) return;
    [self.owner revealSourceAt:location];
}
- (NSMenu *)menuForEvent:(NSEvent *)event {
    NSMenu *menu=[[super menuForEvent:event] copy] ?: [NSMenu new];
    NSPoint point=[self convertPoint:event.locationInWindow fromView:nil];
    NSUInteger location=[self characterIndexForInsertionAtPoint:point];
    if (!NSLocationInRange(location,self.selectedRange)) self.selectedRange=NSMakeRange(location,0);
    BOOL table=CRCellAt(self.owner,location)!=nil;
    BOOL list=CRList(CRParagraphAt(self.owner,location));
    if ([self.owner imageFigureAt:location]) {
        [menu addItem:NSMenuItem.separatorItem];
        NSMenuItem *title=[[NSMenuItem alloc] initWithTitle:@"图片大小" action:NULL keyEquivalent:@""];
        [menu addItem:title];
        for (NSArray *entry in @[
            @[@"原始大小", @""], @[@"25%", @"25%"], @[@"50%", @"50%"], @[@"75%", @"75%"], @[@"100%", @"100%"],
            @[@"400 像素宽", @"400x"], @[@"600 像素宽", @"600x"], @[@"800 像素宽", @"800x"],
        ]) {
            NSMenuItem *item=[[NSMenuItem alloc] initWithTitle:entry[0] action:@selector(structuralAction:) keyEquivalent:@""];
            item.target=self; item.representedObject=[@"image-size:" stringByAppendingString:entry[1]];
            if ([entry[1] isEqual:[self.owner imageSpecAt:location]]) item.state=NSControlStateValueOn;
            [menu addItem:item];
        }
        [menu addItem:NSMenuItem.separatorItem];
        NSMenuItem *custom=[[NSMenuItem alloc] initWithTitle:@"自定义大小…" action:@selector(chooseImageSize:) keyEquivalent:@""];
        custom.target=self; custom.representedObject=@{@"location":@(location),@"revision":@(self.owner.documentRevision)};
        [menu addItem:custom];
        return menu;
    }
    if (CRKind(CRParagraphAt(self.owner,location),@"toc")) {
        [menu addItem:NSMenuItem.separatorItem];
        NSMenuItem *marker=[[NSMenuItem alloc] initWithTitle:@"编辑目录标记" action:@selector(revealTOCMarker:) keyEquivalent:@""];
        marker.target=self; marker.representedObject=@{@"command":@"toc:reveal",@"location":@(location),@"revision":@(self.owner.documentRevision)};
        [menu addItem:marker];
        return menu;
    }
    NSString *footnote=[self.owner footnoteLabelAt:location];
    if (footnote.length) {
        [menu addItem:NSMenuItem.separatorItem];
        for (NSArray *entry in @[@[@"跳到脚注定义",@"footnote:definition"],@[@"返回脚注引用",@"footnote:reference"]]) {
            NSMenuItem *item=[[NSMenuItem alloc] initWithTitle:entry[0] action:@selector(footnoteAction:) keyEquivalent:@""];
            item.target=self; item.representedObject=@{@"command":entry[1],@"location":@(location),@"revision":@(self.owner.documentRevision),@"label":footnote};
            [menu addItem:item];
        }
    }
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
    if (item.action==@selector(footnoteAction:)) return !self.hasMarkedText;
    if (item.action==@selector(structuralAction:) || item.action==@selector(chooseCodeLanguage:) || item.action==@selector(codeLanguageAction:) || item.action==@selector(revealTOCMarker:)) return self.editable && !self.hasMarkedText;
    return [super validateMenuItem:item];
}

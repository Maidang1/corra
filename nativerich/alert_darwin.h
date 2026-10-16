// GitHub-style callouts (alerts): a blockquote whose first line is [!NOTE],
// [!TIP], [!IMPORTANT], [!WARNING] or [!CAUTION] renders as a tinted card
// instead of a plain quote. The marker stays in the document as ordinary text,
// so the native editor keeps editing it in place and only changes how it is
// painted. Everything here is presentation plus the metadata that carries the
// label; the serializer never sees a different document.

static NSSet *CRAlertLabels(void) {
    static NSSet *labels = nil;
    static dispatch_once_t once;
    dispatch_once(&once, ^{
        labels = [NSSet setWithObjects:@"NOTE", @"TIP", @"IMPORTANT", @"WARNING", @"CAUTION", nil];
    });
    return labels;
}

// CRAlertLabelOf reads a "[!LABEL]" marker off the start of a quote paragraph's
// text. The marker counts both on its own and with text after it, because the
// paragraph's text merges the marker line with the lines under it.
static NSString *CRAlertLabelOf(NSString *text) {
    if (![text hasPrefix:@"[!"]) return nil;
    NSRange close = [text rangeOfString:@"]"];
    if (close.location == NSNotFound || close.location < 2) return nil;
    NSString *label = [text substringWithRange:NSMakeRange(2, close.location - 2)];
    if (![CRAlertLabels() containsObject:label.uppercaseString]) return nil;
    NSString *rest = [text substringFromIndex:NSMaxRange(close)];
    if (rest.length && ![rest hasPrefix:@" "] && ![rest hasPrefix:@"\t"] && ![rest hasPrefix:@"\u2028"]) return nil;
    return label.uppercaseString;
}

// CRStripAlertMarker drops a leading "[!LABEL] " from a paragraph's text, so a
// different label can be written in its place.
static NSString *CRStripAlertMarker(NSString *text) {
    if (CRAlertLabelOf(text) == nil) return text;
    NSRange close = [text rangeOfString:@"]"];
    NSString *rest = [text substringFromIndex:NSMaxRange(close)];
    NSRange gap = [rest rangeOfCharacterFromSet:[NSCharacterSet characterSetWithCharactersInString:@" \t\u2028"]];
    if (gap.location == 0) rest = [rest substringFromIndex:NSMaxRange(gap)];
    return rest;
}

// CRAlertContinues reports whether a quote paragraph is still inside the
// blockquote that opened the callout. A blank ">" line keeps one blockquote
// together; a blank source line starts a new one.
static BOOL CRAlertContinues(NSDictionary *p) {
    NSString *separator = [p[@"separator"] isKindOfClass:NSString.class] ? p[@"separator"] : @"";
    return [separator rangeOfString:@">"].location != NSNotFound;
}

static NSColor *CRAlertAccent(NSString *label, BOOL dark) {
    struct { NSString *name; CGFloat r, g, b; } palette[] = {
        {@"NOTE", 0.33, 0.62, 1.00}, {@"TIP", 0.29, 0.72, 0.42},
        {@"IMPORTANT", 0.69, 0.38, 0.96}, {@"WARNING", 0.83, 0.65, 0.17},
        {@"CAUTION", 1.00, 0.47, 0.47},
    };
    CGFloat r = 0.33, g = 0.62, b = 1.00;
    for (unsigned i = 0; i < sizeof(palette) / sizeof(palette[0]); i++) {
        if ([label isEqual:palette[i].name]) { r = palette[i].r; g = palette[i].g; b = palette[i].b; break; }
    }
    return [NSColor colorWithRed:r green:g blue:b alpha:dark ? 0.95 : 1.0];
}

// CRAlertTint washes the callout's background. It is applied as a text
// attribute rather than drawn, so it follows the text when it reflows.
static NSColor *CRAlertTint(NSString *label, BOOL dark) {
    return [CRAlertAccent(label, dark) colorWithAlphaComponent:dark ? 0.16 : 0.13];
}

// refreshAlerts re-derives the callout label of every quote paragraph. The
// marker is ordinary text, so typing or deleting one changes the label without
// anything else noticing.
- (void)refreshAlerts {
    NSString *carried = nil;
    for (NSValue *value in CRRanges(_text.string)) {
        NSRange range = value.rangeValue;
        NSDictionary *p = CRParagraphAt(self, range.location);
        NSString *label = nil;
        if (CRKind(p, @"quote")) {
            NSString *text = range.length ? [_text.string substringWithRange:CRContent(_text.string, range)] : @"";
            label = carried.length ? carried : CRAlertLabelOf(text);
            carried = (label.length && CRAlertContinues(p)) ? label : nil;
        } else {
            carried = nil;
        }
        NSString *current = [p[@"alert"] isKindOfClass:NSString.class] ? p[@"alert"] : nil;
        if ((current ?: @"").length == 0 && !label) continue;
        if ([current isEqual:label ?: @""]) continue;
        NSMutableDictionary *updated = [p mutableCopy];
        if (label.length) updated[@"alert"] = label;
        else [updated removeObjectForKey:@"alert"];
        if (range.length) { [_text.textStorage addAttribute:CRParagraph value:updated range:range]; [self markDirty:range]; }
        else {
            NSMutableDictionary *typing = [_text.typingAttributes mutableCopy];
            typing[CRParagraph] = updated;
            _text.typingAttributes = typing;
            _emptyParagraph = updated;
        }
    }
}

// setQuoteAlert writes (or, for an empty label, removes) the callout marker at
// the start of the paragraph the caret is in, turning it into a quote first if
// it is not one yet.
- (void)setQuoteAlert:(NSString *)label {
    if (!_text.editable || _text.hasMarkedText || _text.selectedRange.length) return;
    label = label.uppercaseString;
    if (label.length && ![CRAlertLabels() containsObject:label]) return;
    NSUInteger caret = _text.selectedRange.location;
    NSRange paragraph = [_text.string paragraphRangeForRange:NSMakeRange(caret, 0)];
    NSDictionary *p = CRParagraphAt(self, paragraph.location);
    if (!CRKind(p, @"quote")) { [self format:@"quote"]; [self refreshAlerts]; }
    p = CRParagraphAt(self, paragraph.location);
    if (!CRKind(p, @"quote")) return;
    [self beginStructuralCommand];
    NSMutableAttributedString *after = [_text.textStorage mutableCopy];
    NSRange content = CRContent(_text.string, paragraph);
    NSString *text = content.length ? [_text.string substringWithRange:content] : @"";
    NSString *updated = CRStripAlertMarker(text);
    if (label.length) updated = [[NSString stringWithFormat:@"[!%@] ", label] stringByAppendingString:updated];
    NSDictionary *attrs = content.length
        ? [_text.textStorage attributesAtIndex:content.location effectiveRange:NULL]
        : _text.typingAttributes;
    [after replaceCharactersInRange:(content.length ? content : paragraph)
              withAttributedString:[[NSAttributedString alloc] initWithString:updated attributes:attrs]];
    NSUInteger caretAfter = paragraph.location + updated.length;
    [self replace:NSMakeRange(0, _text.string.length) with:after selection:NSMakeRange(caretAfter, 0)];
    [_text scrollRangeToVisible:_text.selectedRange];
}

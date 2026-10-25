// Typora sizes an image with a suffix on its destination: "=300x200", "=300x",
// "=x200" or "=50%". The preview and the exporters already read it; the native
// editor has to as well, or it draws every figure at its natural size and drops
// the suffix the document asks for. Everything here reads that suffix, applies
// it to the attachment's bounds and rewrites it in the source when the reader
// picks a new size.

// CRImageDigits reports whether every character of s is a decimal digit.
static BOOL CRImageDigits(NSString *s) {
    if (!s.length) return NO;
    NSCharacterSet *digits = NSCharacterSet.decimalDigitCharacterSet;
    for (NSUInteger i = 0; i < s.length; i++) {
        if (![digits characterIsMember:[s characterAtIndex:i]]) return NO;
    }
    return YES;
}

// CRImageSizeValid recognizes the four size forms and rejects anything else, so
// a "=" in a query string is not mistaken for a size.
static BOOL CRImageSizeValid(NSString *spec) {
    if (!spec.length) return NO;
    if ([spec hasSuffix:@"%"]) return CRImageDigits([spec substringToIndex:spec.length - 1]);
    NSArray<NSString *> *parts = [spec componentsSeparatedByString:@"x"];
    if (parts.count > 2) return NO;
    for (NSString *part in parts) {
        if (part.length && !CRImageDigits(part)) return NO;
    }
    return parts[0].length || parts[1].length;
}

// CRImageDestinationParts splits a destination into the URL, the quoted title
// and the size suffix. The suffix only counts when it follows the URL and any
// title after whitespace, so the "=" of a query string is left alone.
static void CRImageDestinationParts(NSString *destination, NSString **url, NSString **title, NSString **spec) {
    *url = destination ?: @""; *title = @""; *spec = nil;
    NSString *rest = destination ?: @"";
    if ([rest hasPrefix:@"<"]) {
        // An angle-bracketed destination keeps the spaces that would end a bare one.
        NSRange close = [rest rangeOfString:@">"];
        if (close.location == NSNotFound) return;
        *url = [rest substringWithRange:NSMakeRange(1, close.location - 1)];
        rest = [[rest substringFromIndex:NSMaxRange(close)] stringByTrimmingCharactersInSet:NSCharacterSet.whitespaceCharacterSet];
    } else {
        NSRange space = [rest rangeOfCharacterFromSet:NSCharacterSet.whitespaceCharacterSet];
        if (space.location == NSNotFound) return;
        *url = [rest substringToIndex:space.location];
        rest = [[rest substringFromIndex:space.location] stringByTrimmingCharactersInSet:NSCharacterSet.whitespaceCharacterSet];
    }
    if (!rest.length) return;
    unichar quote = [rest characterAtIndex:0];
    if (quote == '"' || quote == '\'') {
        // The title is kept with its quotes so it can be written back verbatim.
        NSRange close = [rest rangeOfString:[NSString stringWithCharacters:&quote length:1] options:0 range:NSMakeRange(1, rest.length - 1)];
        if (close.location == NSNotFound) return;
        *title = [rest substringWithRange:NSMakeRange(0, NSMaxRange(close))];
        rest = [[rest substringFromIndex:NSMaxRange(close)] stringByTrimmingCharactersInSet:NSCharacterSet.whitespaceCharacterSet];
        if (![rest hasPrefix:@"="]) return;
    } else if (![rest hasPrefix:@"="]) {
        return;
    }
    NSString *candidate = [rest substringFromIndex:1];
    if (CRImageSizeValid(candidate)) *spec = candidate;
}

// CRImageSizeSpec reads the size suffix off a destination, without the "=".
static NSString *CRImageSizeSpec(NSString *destination) {
    NSString *url = nil, *title = nil, *spec = nil;
    CRImageDestinationParts(destination, &url, &title, &spec);
    return spec;
}

// CRImageDestination reads the file URL out of an image destination, leaving
// the title and the size suffix behind.
static NSString *CRImageDestination(NSString *destination) {
    NSString *url = nil, *title = nil, *spec = nil;
    CRImageDestinationParts(destination, &url, &title, &spec);
    return url;
}

// CRImageSizedDestination writes spec back into a destination, replacing the
// suffix that was there. An empty spec removes it, which asks for the natural
// size again.
static NSString *CRImageSizedDestination(NSString *destination, NSString *spec) {
    NSString *url = nil, *title = nil, *previous = nil;
    CRImageDestinationParts(destination, &url, &title, &previous);
    NSString *out = url;
    // A URL holding whitespace only survives inside angle brackets.
    if ([out rangeOfCharacterFromSet:NSCharacterSet.whitespaceCharacterSet].location != NSNotFound) {
        out = [NSString stringWithFormat:@"<%@>", out];
    }
    if (title.length) out = [out stringByAppendingFormat:@" %@", title];
    if (spec.length) out = [out stringByAppendingFormat:@" =%@", spec];
    return out;
}

// CRImageSizedRaw rewrites the size suffix of an image's source.
static NSString *CRImageSizedRaw(NSString *raw, NSString *spec) {
    if (![raw hasPrefix:@"!["]) return raw;
    NSRange open = [raw rangeOfString:@"]("];
    if (open.location == NSNotFound) return raw;
    NSUInteger start = NSMaxRange(open);
    NSRange close = [raw rangeOfString:@")" options:0 range:NSMakeRange(start, raw.length - start)];
    if (close.location == NSNotFound) return raw;
    NSString *destination = [raw substringWithRange:NSMakeRange(start, close.location - start)];
    NSString *updated = CRImageSizedDestination(destination, spec);
    if ([updated isEqual:destination]) return raw;
    return [raw stringByReplacingCharactersInRange:NSMakeRange(start, close.location - start) withString:updated];
}

// CRImageReplacedRaw swaps the URL of an image's source, keeping the quoted
// title and the size suffix that were there. A URL holding whitespace is
// wrapped in angle brackets, the way CRImageSizedDestination writes it.
static NSString *CRImageReplacedRaw(NSString *raw, NSString *url) {
    if (![raw hasPrefix:@"!["]) return raw;
    NSRange open = [raw rangeOfString:@"]("];
    if (open.location == NSNotFound) return raw;
    NSUInteger start = NSMaxRange(open);
    NSRange close = [raw rangeOfString:@")" options:0 range:NSMakeRange(start, raw.length - start)];
    if (close.location == NSNotFound) return raw;
    NSString *destination = [raw substringWithRange:NSMakeRange(start, close.location - start)];
    NSString *previous = nil, *title = nil, *spec = nil;
    CRImageDestinationParts(destination, &previous, &title, &spec);
    NSString *updated = url;
    if ([updated rangeOfCharacterFromSet:NSCharacterSet.whitespaceCharacterSet].location != NSNotFound) {
        updated = [NSString stringWithFormat:@"<%@>", updated];
    }
    if (title.length) updated = [updated stringByAppendingFormat:@" %@", title];
    if (spec.length) updated = [updated stringByAppendingFormat:@" =%@", spec];
    if ([updated isEqual:destination]) return raw;
    return [raw stringByReplacingCharactersInRange:NSMakeRange(start, close.location - start) withString:updated];
}

// CRImagePart reads one side of a "WxH" spec; an absent side stays 0.
static CGFloat CRImagePart(NSString *s) { return s.length ? (CGFloat)s.integerValue : 0; }

// CRImageBoundsFromSpec sizes a figure the way its document asks. A width and
// height are pixels, one side alone keeps the aspect ratio, and a percentage
// scales the natural size. maxWidth is the column the figure has to fit in.
static NSRect CRImageBoundsFromSpec(NSImage *img, NSString *spec, CGFloat maxWidth) {
    CGFloat w = img.size.width, h = img.size.height;
    if (w <= 0 || h <= 0) return NSMakeRect(0, 0, 1, 1);
    if (spec.length) {
        if ([spec hasSuffix:@"%"]) {
            CGFloat percent = CRImagePart([spec substringToIndex:spec.length - 1]);
            if (percent > 0) { w = w * percent / 100; h = h * percent / 100; }
        } else {
            NSArray<NSString *> *parts = [spec componentsSeparatedByString:@"x"];
            CGFloat pw = CRImagePart(parts[0]);
            CGFloat ph = parts.count > 1 ? CRImagePart(parts[1]) : 0;
            if (pw > 0 && ph > 0) { w = pw; h = ph; }
            else if (pw > 0) { h = h * pw / w; w = pw; }
            else if (ph > 0) { w = w * ph / h; h = ph; }
        }
    }
    if (w > maxWidth) { h = h * maxWidth / w; w = maxWidth; }
    return NSMakeRect(0, 0, w, h);
}

// imageFigureRanges lists the image paragraphs whose figure is drawn, which is
// the only case a size can be applied to: a figure the reader typed into is
// source text now.
- (NSArray *)imageFigureRanges {
    NSMutableArray *figures = [NSMutableArray array];
    for (NSValue *value in CRRanges(_text.string)) {
        NSRange range = value.rangeValue;
        NSRange content = CRContent(_text.string, range);
        if (!CRKind(CRParagraphAt(self, range.location), @"image")) continue;
        if (content.length != 1 || ![[_text.string substringWithRange:content] isEqual:@"\uFFFC"]) continue;
        [figures addObject:value];
    }
    return figures;
}

// CRImageRawDestination returns the destination of an image's source, which is
// where the URL, the title and the size suffix all live.
static NSString *CRImageRawDestination(NSString *raw) {
    if (![raw hasPrefix:@"!["]) return nil;
    NSRange open = [raw rangeOfString:@"]("];
    if (open.location == NSNotFound) return nil;
    NSUInteger start = NSMaxRange(open);
    NSRange close = [raw rangeOfString:@")" options:0 range:NSMakeRange(start, raw.length - start)];
    if (close.location == NSNotFound) return nil;
    return [raw substringWithRange:NSMakeRange(start, close.location - start)];
}

// CRImageRawSpec reads the size suffix out of an image's source.
static NSString *CRImageRawSpec(NSString *raw) {
    NSString *destination = CRImageRawDestination(raw);
    return destination ? CRImageSizeSpec(destination) : nil;
}

// CRImageRawURL reads the file URL out of an image's source.
static NSString *CRImageRawURL(NSString *raw) {
    NSString *destination = CRImageRawDestination(raw);
    return destination ? CRImageDestination(destination) : nil;
}

- (NSDictionary *)imageFigureAt:(NSUInteger)caret {
    for (NSValue *value in [self imageFigureRanges].reverseObjectEnumerator) {
        NSRange range = value.rangeValue;
        if (NSLocationInRange(caret, range) || (caret == NSMaxRange(range) && caret == _text.string.length)) {
            return CRParagraphAt(self, range.location);
        }
    }
    return nil;
}

- (NSString *)imageSpecAt:(NSUInteger)caret {
    NSDictionary *p = [self imageFigureAt:caret];
    if (!p) return nil;
    NSString *raw = [p[@"raw"] isKindOfClass:NSString.class] ? p[@"raw"] : @"";
    return CRImageRawSpec(raw) ?: @"";
}

- (void)setImageSize:(NSString *)spec {
    [self setImageSize:spec atLocation:_text.selectedRange.location];
}

// setImageSize rewrites the size suffix of the image paragraph at caret and
// redraws its figure. The document keeps asking for the new size, so the
// preview and the exports agree with what the reader sees.
- (void)setImageSize:(NSString *)spec atLocation:(NSUInteger)caret {
    if (!_text.editable || _text.hasMarkedText || ![spec isKindOfClass:NSString.class]) return;
    if (spec.length && !CRImageSizeValid(spec)) return;
    NSDictionary *found = [self imageFigureAt:caret];
    if (!found) return;
    NSMutableDictionary *paragraph = [found mutableCopy];
    NSString *raw = [paragraph[@"raw"] isKindOfClass:NSString.class] ? paragraph[@"raw"] : @"";
    NSString *updated = CRImageSizedRaw(raw, spec);
    if ([updated isEqual:raw]) return;
    paragraph[@"raw"] = updated;
    NSMutableAttributedString *after = [_text.textStorage mutableCopy];
    NSRange content = CRContent(_text.string, [_text.string paragraphRangeForRange:NSMakeRange(caret, 0)]);
    [after addAttribute:CRParagraph value:paragraph range:content];
    // The figure is an attachment, so the new size has to be drawn and not
    // only recorded.
    NSAttributedString *figure = [self figure:paragraph];
    NSTextAttachment *attachment = figure.length ? [figure attribute:NSAttachmentAttributeName atIndex:0 effectiveRange:NULL] : nil;
    if (attachment) [after addAttribute:NSAttachmentAttributeName value:attachment range:content];
    [self beginStructuralCommand];
    [_undo beginUndoGrouping];
    [self restore:after selection:_text.selectedRange typing:_text.typingAttributes ending:_emptyParagraph];
    [_undo endUndoGrouping];
}

// replaceImageAtLocation swaps the figure's file for another URL, keeping the
// alt text, the quoted title and the size suffix, and redraws the figure the
// same way setImageSize does. The reader picked a new picture, not a new
// layout, so everything around the URL survives.
- (BOOL)replaceImageAtLocation:(NSUInteger)caret withURL:(NSString *)url {
    if (!_text.editable || _text.hasMarkedText || !url.length) return NO;
    // A file inside the document's folder is written relative, so the note
    // survives being moved with its assets.
    if (_baseDir.length && [url hasPrefix:[_baseDir stringByAppendingString:@"/"]]) {
        url = [url substringFromIndex:_baseDir.length + 1];
    }
    NSDictionary *found = [self imageFigureAt:caret];
    if (!found) return NO;
    NSMutableDictionary *paragraph = [found mutableCopy];
    NSString *raw = [paragraph[@"raw"] isKindOfClass:NSString.class] ? paragraph[@"raw"] : @"";
    NSString *updated = CRImageReplacedRaw(raw, url);
    if ([updated isEqual:raw]) return NO;
    paragraph[@"raw"] = updated;
    NSMutableAttributedString *after = [_text.textStorage mutableCopy];
    NSRange content = CRContent(_text.string, [_text.string paragraphRangeForRange:NSMakeRange(caret, 0)]);
    [after addAttribute:CRParagraph value:paragraph range:content];
    NSAttributedString *figure = [self figure:paragraph];
    NSTextAttachment *attachment = figure.length ? [figure attribute:NSAttachmentAttributeName atIndex:0 effectiveRange:NULL] : nil;
    if (attachment) [after addAttribute:NSAttachmentAttributeName value:attachment range:content];
    [self beginStructuralCommand];
    [_undo beginUndoGrouping];
    [self restore:after selection:_text.selectedRange typing:_text.typingAttributes ending:_emptyParagraph];
    [_undo endUndoGrouping];
    return YES;
}

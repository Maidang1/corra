// Footnote hover card: a small readable popover for a footnote reference, so
// the definition does not wait for the system tooltip. Included at file scope;
// the editor half lives in hover_controls_darwin.h.

#pragma once

@class CREditor;

static const CGFloat CRFootnoteCardWidth = 300;

@interface CRFootnoteCardPanel : NSPanel
@end
@implementation CRFootnoteCardPanel
- (BOOL)canBecomeKey { return NO; }
- (BOOL)canBecomeMain { return NO; }
@end

@interface CRFootnoteCardView : NSView
@property(nonatomic, weak) CREditor *owner;
@property(nonatomic, copy) NSString *heading;
@property(nonatomic, copy) NSString *body;
// showsJumpButton adds the "跳到定义" link under the body; it only makes
// sense when the hovered reference has a definition to land on.
@property(nonatomic) BOOL showsJumpButton;
// referenceLocation is the hovered mark's character index, the jump's label
// source and the spot Cmd-Opt-Left returns to afterwards.
@property(nonatomic) NSUInteger referenceLocation;
@end

@implementation CRFootnoteCardView {
    NSButton *_jumpButton;
    NSTrackingArea *_tracking;
}
- (BOOL)isFlipped { return YES; }
- (NSDictionary *)titleAttrs {
    BOOL dark = self.owner.dark;
    return @{NSFontAttributeName: [NSFont systemFontOfSize:11 weight:NSFontWeightSemibold],
             NSForegroundColorAttributeName: dark ? [NSColor colorWithWhite:0.62 alpha:1] : [NSColor colorWithWhite:0.42 alpha:1]};
}
- (NSDictionary *)bodyAttrs {
    BOOL dark = self.owner.dark;
    NSMutableParagraphStyle *style = [NSMutableParagraphStyle new];
    style.lineSpacing = 2;
    return @{NSFontAttributeName: [NSFont systemFontOfSize:12.5],
             NSForegroundColorAttributeName: dark ? [NSColor colorWithWhite:0.90 alpha:1] : [NSColor colorWithWhite:0.18 alpha:1],
             NSParagraphStyleAttributeName: style};
}
- (CGFloat)cardHeight {
    CGFloat width = CRFootnoteCardWidth - 24;
    NSRect text = [self.body boundingRectWithSize:NSMakeSize(width, CGFLOAT_MAX)
                                          options:NSStringDrawingUsesLineFragmentOrigin
                                       attributes:[self bodyAttrs]];
    CGFloat height = 12 + 15 + 6 + MIN(ceil(text.size.height), 120) + 12;
    return self.showsJumpButton ? height + 24 : height;
}
- (void)setShowsJumpButton:(BOOL)shows {
    _showsJumpButton = shows;
    if (shows && !_jumpButton) {
        _jumpButton = [NSButton new];
        _jumpButton.bordered = NO;
        _jumpButton.font = [NSFont systemFontOfSize:11 weight:NSFontWeightMedium];
        NSMutableAttributedString *title = [[NSMutableAttributedString alloc] initWithString:@"跳到定义 ›"];
        [title addAttribute:NSForegroundColorAttributeName value:NSColor.linkColor range:NSMakeRange(0, title.length)];
        _jumpButton.attributedTitle = title;
        _jumpButton.target = self;
        _jumpButton.action = @selector(jumpToDefinition:);
        [self addSubview:_jumpButton];
    }
    _jumpButton.hidden = !shows;
    self.needsLayout = YES;
}
- (void)layout {
    [super layout];
    if (_jumpButton) {
        [_jumpButton sizeToFit];
        _jumpButton.frame = NSMakeRect(10, self.bounds.size.height - 24, _jumpButton.frame.size.width, 18);
    }
}
- (void)jumpToDefinition:(id)sender {
    [self.owner.text footnoteCardJump];
}
// The panel keeps the card open while the pointer is over it; leaving the
// card dismisses it, as leaving the reference does.
- (void)updateTrackingAreas {
    [super updateTrackingAreas];
    if (_tracking) [self removeTrackingArea:_tracking];
    _tracking = [[NSTrackingArea alloc] initWithRect:self.bounds
                                             options:NSTrackingMouseEnteredAndExited | NSTrackingActiveAlways
                                               owner:self userInfo:nil];
    [self addTrackingArea:_tracking];
}
- (void)mouseExited:(NSEvent *)event {
    [self.owner.text hideFootnoteCard];
}
- (void)drawRect:(NSRect)dirty {
    BOOL dark = self.owner.dark;
    NSRect bounds = self.bounds;
    NSBezierPath *outline = [NSBezierPath bezierPathWithRoundedRect:NSInsetRect(bounds, 0.5, 0.5) xRadius:10 yRadius:10];
    [outline addClip];
    [(dark ? [NSColor colorWithWhite:0.18 alpha:1] : [NSColor whiteColor]) setFill];
    [NSBezierPath fillRect:bounds];
    [[(dark ? [NSColor whiteColor] : [NSColor blackColor]) colorWithAlphaComponent:0.13] setStroke];
    [outline setLineWidth:1];
    [outline stroke];
    [self.heading drawAtPoint:NSMakePoint(12, 12) withAttributes:[self titleAttrs]];
    CGFloat width = bounds.size.width - 24;
    NSRect bodyRect = NSMakeRect(12, 33, width, MIN(120, bounds.size.height - 45));
    [self.body drawWithRect:bodyRect
                    options:NSStringDrawingUsesLineFragmentOrigin | NSStringDrawingTruncatesLastVisibleLine
                 attributes:[self bodyAttrs]];
}
@end

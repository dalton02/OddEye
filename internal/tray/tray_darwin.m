#import <Cocoa/Cocoa.h>

extern void goTrayClicked(int id);

@interface PRGTrayTarget : NSObject
- (void)clicked:(NSMenuItem *)sender;
@end

@implementation PRGTrayTarget
- (void)clicked:(NSMenuItem *)sender {
    goTrayClicked((int)sender.tag);
}
@end

static NSStatusItem *statusItem;
static NSMenu *menu;
static PRGTrayTarget *target;
static NSImage *iconNormal;
static NSImage *iconDot;

static NSImage *makeIcon(NSData *data) {
    NSImage *img = [[NSImage alloc] initWithData:data];
    [img setSize:NSMakeSize(18, 18)];
    [img setTemplate:YES];
    return img;
}

// Todas as funções copiam os dados de Go antes do dispatch: a memória do chamador é liberada assim
// que a função retorna, e o AppKit só pode ser tocado na thread principal (onde roda o loop do Wails).
void trayStart(const void *normal, int normalLen, const void *dot, int dotLen,
               const int *ids, const char **titles, const int *visible, int count) {
    @autoreleasepool {
        NSData *normalData = [NSData dataWithBytes:normal length:normalLen];
        NSData *dotData = [NSData dataWithBytes:dot length:dotLen];
        NSMutableArray *entries = [NSMutableArray arrayWithCapacity:count];
        for (int i = 0; i < count; i++) {
            [entries addObject:@{
                @"id": @(ids[i]),
                @"title": [NSString stringWithUTF8String:titles[i]],
                @"visible": @(visible[i])
            }];
        }
        dispatch_async(dispatch_get_main_queue(), ^{
            iconNormal = makeIcon(normalData);
            iconDot = makeIcon(dotData);
            target = [[PRGTrayTarget alloc] init];
            menu = [[NSMenu alloc] init];
            for (NSDictionary *e in entries) {
                int itemId = [e[@"id"] intValue];
                if (itemId < 0) {
                    [menu addItem:[NSMenuItem separatorItem]];
                    continue;
                }
                NSMenuItem *item = [[NSMenuItem alloc] initWithTitle:e[@"title"] action:nil keyEquivalent:@""];
                item.tag = itemId;
                // O item de status (id 0) fica sem ação: o NSMenu o desabilita sozinho.
                if (itemId != 0) {
                    item.target = target;
                    item.action = @selector(clicked:);
                }
                item.hidden = ![e[@"visible"] boolValue];
                [menu addItem:item];
            }
            statusItem = [[NSStatusBar systemStatusBar] statusItemWithLength:NSSquareStatusItemLength];
            statusItem.button.image = iconNormal;
            statusItem.menu = menu;
        });
    }
}

void traySetItem(int itemId, const char *title, int visible) {
    @autoreleasepool {
        NSString *t = [NSString stringWithUTF8String:title];
        dispatch_async(dispatch_get_main_queue(), ^{
            NSMenuItem *item = [menu itemWithTag:itemId];
            if (item == nil) return;
            if (visible) item.title = t;
            item.hidden = !visible;
        });
    }
}

void traySetIcon(int dot) {
    dispatch_async(dispatch_get_main_queue(), ^{
        if (statusItem == nil) return;
        statusItem.button.image = dot ? iconDot : iconNormal;
    });
}

void traySetTooltip(const char *text) {
    @autoreleasepool {
        NSString *t = [NSString stringWithUTF8String:text];
        dispatch_async(dispatch_get_main_queue(), ^{
            if (statusItem == nil) return;
            statusItem.button.toolTip = t;
        });
    }
}

void trayStop(void) {
    dispatch_async(dispatch_get_main_queue(), ^{
        if (statusItem == nil) return;
        [[NSStatusBar systemStatusBar] removeStatusItem:statusItem];
        statusItem = nil;
    });
}

import CoreGraphics
import Foundation
import ImageIO

// 将原始图标裁成带透明圆角的方形，保留人物和白色背景，只处理四个角。
func renderRoundedIcon(input: URL, output: URL, size: Int, radius: CGFloat) throws {
    guard let source = CGImageSourceCreateWithURL(input as CFURL, nil),
          let image = CGImageSourceCreateImageAtIndex(source, 0, nil) else {
        throw NSError(domain: "RoundIcon", code: 1, userInfo: [NSLocalizedDescriptionKey: "无法读取输入图标"])
    }

    let canvas = CGRect(x: 0, y: 0, width: size, height: size)
    guard let context = CGContext(
        data: nil,
        width: size,
        height: size,
        bitsPerComponent: 8,
        bytesPerRow: size * 4,
        space: CGColorSpaceCreateDeviceRGB(),
        bitmapInfo: CGImageAlphaInfo.premultipliedLast.rawValue
    ) else {
        throw NSError(domain: "RoundIcon", code: 2, userInfo: [NSLocalizedDescriptionKey: "无法创建图标画布"])
    }

    // 透明画布配合圆角裁剪，让 Finder 和顶部栏都不再显示硬直角。
    context.clear(canvas)
    let roundedRect = UIBezierPathLike.roundedRect(canvas, radius: radius)
    context.addPath(roundedRect)
    context.clip()
    context.draw(image, in: canvas)

    guard let destination = CGImageDestinationCreateWithURL(
        output as CFURL,
        "public.png" as CFString,
        1,
        nil
    ), let rendered = context.makeImage() else {
        throw NSError(domain: "RoundIcon", code: 3, userInfo: [NSLocalizedDescriptionKey: "无法保存圆角图标"])
    }
    CGImageDestinationAddImage(destination, rendered, nil)
    guard CGImageDestinationFinalize(destination) else {
        throw NSError(domain: "RoundIcon", code: 4, userInfo: [NSLocalizedDescriptionKey: "保存圆角图标失败"])
    }
}

// 使用 CoreGraphics 构造圆角路径，避免引入 AppKit 或第三方依赖。
enum UIBezierPathLike {
    static func roundedRect(_ rect: CGRect, radius: CGFloat) -> CGPath {
        CGPath(roundedRect: rect, cornerWidth: radius, cornerHeight: radius, transform: nil)
    }
}

// 命令行入口：生成 App 图标和顶部栏小图标两种尺寸。
let arguments = CommandLine.arguments
guard arguments.count == 5,
      let size = Int(arguments[3]),
      let radius = Double(arguments[4]) else {
    throw NSError(domain: "RoundIcon", code: 5, userInfo: [NSLocalizedDescriptionKey: "用法：round_icon 输入.png 输出.png 尺寸 圆角"])
}

try renderRoundedIcon(
    input: URL(fileURLWithPath: arguments[1]),
    output: URL(fileURLWithPath: arguments[2]),
    size: size,
    radius: CGFloat(radius)
)

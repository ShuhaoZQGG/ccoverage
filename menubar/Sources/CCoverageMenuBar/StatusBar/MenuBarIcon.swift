import AppKit

enum MenuBarIcon {
    /// Returns the CC logo as an 18×18 template NSImage.
    static func logo() -> NSImage {
        let size = NSSize(width: 18, height: 18)
        let image = NSImage(size: size, flipped: false) { rect in
            guard let ctx = NSGraphicsContext.current?.cgContext else { return false }

            ctx.setLineCap(.round)
            ctx.setStrokeColor(NSColor.black.cgColor)

            // Outer C: center shifted right so the C-opening faces right
            let outerCenter = CGPoint(x: 10, y: 9)
            let outerRadius: CGFloat = 7.0
            ctx.setLineWidth(2.0)
            // Arc from 50° to 310° (skip the right side so opening faces right)
            let outerStart = CGFloat(50) * .pi / 180
            let outerEnd = CGFloat(310) * .pi / 180
            ctx.addArc(center: outerCenter, radius: outerRadius,
                       startAngle: outerStart, endAngle: outerEnd, clockwise: false)
            ctx.strokePath()

            // Inner C: same center, smaller radius
            let innerRadius: CGFloat = 3.5
            ctx.setLineWidth(1.5)
            let innerStart = CGFloat(50) * .pi / 180
            let innerEnd = CGFloat(310) * .pi / 180
            ctx.addArc(center: outerCenter, radius: innerRadius,
                       startAngle: innerStart, endAngle: innerEnd, clockwise: false)
            ctx.strokePath()

            return true
        }
        image.isTemplate = true
        return image
    }
}

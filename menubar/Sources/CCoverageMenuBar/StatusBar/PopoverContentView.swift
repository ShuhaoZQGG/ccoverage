import SwiftUI

struct PopoverContentView: View {
    @ObservedObject var viewModel: DashboardViewModel

    var body: some View {
        DashboardView(viewModel: viewModel)
            .frame(width: 360)
            .fixedSize(horizontal: false, vertical: true)
            .background(.clear)
    }
}

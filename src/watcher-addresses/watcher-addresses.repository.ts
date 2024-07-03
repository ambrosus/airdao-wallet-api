import { singleton } from "tsyringe";
import { WatcherAddressModel } from "./watcher-addresses.model";


@singleton()
export class WatcherAddressesRepository {
    constructor(private readonly model: typeof WatcherAddressModel) {
    }

    async createWatcherAddress(watcherId: string, address: string) {
        return this.model.create({ watcherId, address });
    }

    async getWatcherAddresses(watcherId: string) {
        return this.model.find({ watcherId }).select("-watcherId");
    }

    async getWatcherAddress(watcherId: string, address: string) {
        return this.model.findOne({ watcherId, address });
    }

    async getWatcherIdsByAddress(address: string) {
        return this.model.find({ address }).distinct("watcherId");
    }

    async updateWatcherAddress(watcherId: string, address: string, data: Record<string, unknown>) {
        return this.model.updateOne({ watcherId, address }, data);
    }

    async deleteWatcherAddress(watcherId: string, address: string) {
        return this.model.deleteOne({ watcherId, address });
    }

    async deleteWatcherAddresses(watcherId: string) {
        return this.model.deleteMany({ watcherId });
    }

}
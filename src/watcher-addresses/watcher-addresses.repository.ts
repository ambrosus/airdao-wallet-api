import { singleton } from "tsyringe";
import { WatcherAddressModel } from "./watcher-addresses.model";


@singleton()
export class WatcherAddressesRepository {
    constructor() {
    }

    async createWatcherAddress(watcherId: string, address: string) {
        return WatcherAddressModel.create({ watcherId, address });
    }

    async getAllWatchersAddresses(filter: Record<string, unknown>) {
        return WatcherAddressModel.find(filter);
    }

    async getWatcherAddresses(watcherId: string) {
        return WatcherAddressModel.find({ watcherId }).select("-watcherId");
    }

    async getWatcherAddress(watcherId: string, address: string) {
        return WatcherAddressModel.findOne({ watcherId, address });
    }

    async getUniqueAddresses() {
        return WatcherAddressModel.distinct("address").exec();
    }

    async getWatcherIdsByAddress(address: string) {
        return WatcherAddressModel.find({ address }).distinct("watcherId");
    }

    async updateWatcherAddress(watcherId: string, address: string, data: Record<string, unknown>) {
        return WatcherAddressModel.updateOne({ watcherId, address }, data);
    }

    async deleteWatcherAddress(watcherId: string, address: string) {
        return WatcherAddressModel.deleteOne({ watcherId, address });
    }

    async deleteWatcherAddresses(watcherId: string) {
        return WatcherAddressModel.deleteMany({ watcherId });
    }

}
import { singleton } from "tsyringe";
import { WatcherAddressesRepository } from "./watcher-addresses.repository";
import { Promise } from "mongoose";


@singleton()
export class WatcherAddressesService {

    constructor(private readonly repository: WatcherAddressesRepository) {
    }

    async createWatcherAddress(watcherId: string, address: string) {
        return this.repository.createWatcherAddress(watcherId, address);
    }

    async getAllWatchersAddresses(filter: Record<string, unknown>) {
        return this.repository.getAllWatchersAddresses(filter);
    }

    async getUniqueAddresses() {
        return this.repository.getUniqueAddresses();
    }

    async getWatcherAddresses(watcherId: string) {
        const watcherAddressesDoc =  await this.repository.getWatcherAddresses(watcherId);
        return watcherAddressesDoc.map(addressDoc => addressDoc.address);
    }

    async getWatcherAddressesWithDetails(watcherId: string) {
        return this.repository.getWatcherAddresses(watcherId);
    }

    async getWatcherIdsByAddress(address: string) {
        return this.repository.getWatcherIdsByAddress(address);
    }

    async updateWatcherAddress(watcherId: string, address: string, data: Record<string, unknown>) {
        return this.repository.updateWatcherAddress(watcherId, address, data);
    }

    async deleteWatcherAddresses(watcherId: string, addresses: string[]) {
        console.log("deleteWatcherAddresses", watcherId, addresses);
        const promises = addresses.map(async (address) => {
            await this.repository.deleteWatcherAddress(watcherId, address);
        });

        console.log("promises", promises);

        await Promise.all(promises);

        return true;
    }

    async deleteAllWatcherAddresses(watcherId: string) {
        await this.repository.deleteWatcherAddresses(watcherId);
    }
}
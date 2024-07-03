import {singleton} from "tsyringe";
import {WatcherAddressesRepository} from "./watcher-addresses.repository";
import {Promise} from "mongoose";


@singleton()
export class WatcherAddressesService {

    constructor(private readonly repository: WatcherAddressesRepository) {
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

    async deleteWatcherAddresses(watcherId: string, addresses: string[]) {
        const promises = addresses.map(async (address) => {
            await this.repository.deleteWatcherAddress(watcherId, address);
        });

        await Promise.all(promises);

        return true;
    }

    async deleteAllWatcherAddresses(watcherId: string) {
        await this.repository.deleteWatcherAddresses(watcherId);
    }
}
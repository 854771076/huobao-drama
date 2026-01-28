import request from '../utils/request'
import type { Pose, CreatePoseRequest, UpdatePoseRequest } from '../types/pose'

export const poseAPI = {
    list(dramaId: string | number) {
        return request.get<Pose[]>('/dramas/' + dramaId + '/poses')
    },
    create(data: CreatePoseRequest) {
        return request.post<Pose>('/poses', data)
    },
    update(id: number, data: UpdatePoseRequest) {
        return request.put<void>('/poses/' + id, data)
    },
    delete(id: number) {
        return request.delete<void>('/poses/' + id)
    },
    generateImage(id: number) {
        return request.post<{ task_id: string }>(`/poses/${id}/generate`)
    },
    associateWithStoryboard(storyboardId: number, poseIds: number[]) {
        return request.post<void>(`/storyboards/${storyboardId}/poses`, { pose_ids: poseIds })
    },
    // Extract poses from script
    extractFromScript(episodeId: number|string) {
        return request.post<{ task_id: string }>(`/episodes/${episodeId}/poses/extract`);
    }
}

export interface Pose {
    id: number
    drama_id: number
    name: string
    type?: string
    description?: string
    image_url?: string
    reference_images?: any
    created_at: string
    updated_at: string
}

export interface CreatePoseRequest {
    drama_id: number
    name: string
    type?: string
    description?: string
    image_url?: string
}

export interface UpdatePoseRequest {
    name?: string
    type?: string
    description?: string
    image_url?: string
}

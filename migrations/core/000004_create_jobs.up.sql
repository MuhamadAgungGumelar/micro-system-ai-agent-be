-- Create jobs table for background job queue
CREATE TABLE IF NOT EXISTS jobs (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    client_id UUID NOT NULL,

    -- Job identification
    queue VARCHAR(100) NOT NULL DEFAULT 'default',
    type VARCHAR(100) NOT NULL,
    payload JSONB NOT NULL,

    -- Job status and priority
    status VARCHAR(20) NOT NULL DEFAULT 'pending' CHECK (status IN ('pending', 'processing', 'completed', 'failed', 'retrying', 'cancelled')),
    priority INTEGER NOT NULL DEFAULT 5,

    -- Retry configuration
    attempts INTEGER NOT NULL DEFAULT 0,
    max_retries INTEGER NOT NULL DEFAULT 3,

    -- Scheduling and timing
    scheduled_at TIMESTAMP WITH TIME ZONE,
    started_at TIMESTAMP WITH TIME ZONE,
    completed_at TIMESTAMP WITH TIME ZONE,
    failed_at TIMESTAMP WITH TIME ZONE,

    -- Results and metadata
    error TEXT,
    result JSONB,
    metadata JSONB,

    created_at TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT NOW(),

    -- Foreign key constraint
    CONSTRAINT fk_jobs_client FOREIGN KEY (client_id) REFERENCES clients(id) ON DELETE CASCADE
);

-- Indexes for efficient job processing
CREATE INDEX idx_jobs_client_id ON jobs(client_id);
CREATE INDEX idx_jobs_queue ON jobs(queue);
CREATE INDEX idx_jobs_status ON jobs(status);
CREATE INDEX idx_jobs_priority ON jobs(priority DESC);
CREATE INDEX idx_jobs_created_at ON jobs(created_at);
CREATE INDEX idx_jobs_scheduled_at ON jobs(scheduled_at) WHERE scheduled_at IS NOT NULL;

-- Composite index for job dequeue (most critical query)
-- Finds next pending job in a queue, ordered by priority and creation time
CREATE INDEX idx_jobs_dequeue ON jobs(queue, status, priority DESC, created_at)
    WHERE status = 'pending';

-- Composite index for client job listing
CREATE INDEX idx_jobs_client_status ON jobs(client_id, status, created_at DESC);

-- Trigger to update updated_at timestamp
CREATE OR REPLACE FUNCTION update_jobs_updated_at()
RETURNS TRIGGER AS $$
BEGIN
    NEW.updated_at = NOW();
    RETURN NEW;
END;
$$ LANGUAGE plpgsql;

CREATE TRIGGER trigger_jobs_updated_at
    BEFORE UPDATE ON jobs
    FOR EACH ROW
    EXECUTE FUNCTION update_jobs_updated_at();

-- Comment on table
COMMENT ON TABLE jobs IS 'Background job queue for asynchronous task processing';

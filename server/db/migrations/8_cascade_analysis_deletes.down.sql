ALTER TABLE cannabinoids
  DROP CONSTRAINT cannabinoids_analysis_id_fkey,
  ADD CONSTRAINT cannabinoids_analysis_id_fkey
    FOREIGN KEY (analysis_id)
    REFERENCES analyses(id);

ALTER TABLE terpenes
  DROP CONSTRAINT terpenes_analysis_id_fkey,
  ADD CONSTRAINT terpenes_analysis_id_fkey
    FOREIGN KEY (analysis_id)
    REFERENCES analyses(id);

ALTER TABLE test_summaries
  DROP CONSTRAINT test_summaries_analysis_id_fkey,
  ADD CONSTRAINT test_summaries_analysis_id_fkey
    FOREIGN KEY (analysis_id)
    REFERENCES analyses(id);

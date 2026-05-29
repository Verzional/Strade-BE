import { Entity, PrimaryGeneratedColumn, Column, Unique } from 'typeorm';

@Entity('verification_tokens')
@Unique(['identifier', 'token'])
export class VerificationToken {
  @PrimaryGeneratedColumn('uuid')
  id!: string;

  @Column()
  identifier!: string;

  @Column({ unique: true })
  token!: string;

  @Column({ type: 'timestamp' })
  expires!: Date;
}